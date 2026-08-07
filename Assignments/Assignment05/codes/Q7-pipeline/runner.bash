#!/usr/bin/env bash
#
# runner.bash — a hand-rolled GitLab-CI-like runner.
#
# There is no GitLab Runner here, so this script plays its role: it defines a
# list of stages, executes every job **inside a docker container** (the way a
# runner with the `docker` executor would), keeps a module/build cache in named
# volumes, stores artifacts, fails fast, and prints a pass/fail summary.
#
# usage:
#   ./runner.bash                 # run the whole pipeline
#   ./runner.bash lint test       # run selected jobs only
#   ./runner.bash --list          # show the stages
#   WORKSPACE=/tmp/x ./runner.bash   # run against another checkout
#
set -Eeuo pipefail

# ----------------------------------------------------------------- config ---
PIPELINE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE="${WORKSPACE:-$PIPELINE_ROOT/app}"
CI_DIR="${CI_DIR:-$PIPELINE_ROOT/.ci}"
ARTIFACTS_DIR="$CI_DIR/artifacts"
LOG_DIR="$CI_DIR/logs"

# every job image is pinned — a runner must be reproducible
GO_IMAGE="${GO_IMAGE:-golang:1.23-alpine}"
LINT_IMAGE="${LINT_IMAGE:-golangci/golangci-lint:v1.62.2-alpine}"
DOCKER_IMAGE="${DOCKER_IMAGE:-docker:27-cli}"
SMOKE_IMAGE="${SMOKE_IMAGE:-alpine:3.20}"

# caches survive between pipeline runs, like GitLab's `cache:`
MOD_CACHE_VOL="${MOD_CACHE_VOL:-q7-go-mod-cache}"
BUILD_CACHE_VOL="${BUILD_CACHE_VOL:-q7-go-build-cache}"

APP_IMAGE="${APP_IMAGE:-ci-demo}"
APP_TAG="${APP_TAG:-$(date +%Y%m%d-%H%M%S)}"
DEPLOY_NET="ci-demo-net"
DEPLOY_CONTAINER="ci-demo-staging"

STAGES=(install lint build test deploy)
# jobs whose failure does not stop the pipeline (GitLab's `allow_failure: true`)
ALLOW_FAILURE=()

# ------------------------------------------------------------------ output --
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
    C_RESET=$'\033[0m'; C_DIM=$'\033[2m'; C_RED=$'\033[31m'
    C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'; C_BLUE=$'\033[34m'; C_BOLD=$'\033[1m'
else
    C_RESET=""; C_DIM=""; C_RED=""; C_GREEN=""; C_YELLOW=""; C_BLUE=""; C_BOLD=""
fi

log()      { printf '%s\n' "$*"; }
section()  { printf '\n%s==> %s%s\n' "$C_BOLD$C_BLUE" "$*" "$C_RESET"; }
ok()       { printf '%s  PASS%s  %s\n' "$C_GREEN" "$C_RESET" "$*"; }
bad()      { printf '%s  FAIL%s  %s\n' "$C_RED" "$C_RESET" "$*"; }
warn()     { printf '%s  WARN%s  %s\n' "$C_YELLOW" "$C_RESET" "$*"; }

# --------------------------------------------------------------- docker io --
# Run a shell snippet inside a throw-away container with the workspace mounted
# at /src — the equivalent of one `script:` block of a GitLab job.
in_docker() {
    local image="$1"; shift
    docker run --rm \
        -v "$WORKSPACE:/src" \
        -v "$ARTIFACTS_DIR:/artifacts" \
        -v "$MOD_CACHE_VOL:/go/pkg/mod" \
        -v "$BUILD_CACHE_VOL:/root/.cache/go-build" \
        -w /src \
        -e CGO_ENABLED=0 \
        -e GOFLAGS=-buildvcs=false \
        "$image" sh -eu -c "$*"
}

# Same, but with the docker socket bound in, so the job can drive the daemon
# (this is exactly how a `docker` executor builds images).
in_docker_dind() {
    local image="$1"; shift
    docker run --rm \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v "$WORKSPACE:/src" \
        -v "$ARTIFACTS_DIR:/artifacts" \
        -w /src \
        "$image" sh -eu -c "$*"
}

# ------------------------------------------------------------------- jobs ---
runjob() {
    local stage="$1"

    case "$stage" in
    install)
        # resolve and cache the module graph; nothing else may touch the network
        in_docker "$GO_IMAGE" '
            go mod download -x 2>&1 | tail -5
            go mod verify
            echo "--- modules ---"
            go list -m all
        ' || return 1
        ;;

    lint)
        # gofmt + go vet are the cheap gates, golangci-lint is the thorough one
        in_docker "$GO_IMAGE" '
            unformatted=$(gofmt -l .)
            if [ -n "$unformatted" ]; then
                echo "gofmt found unformatted files:"
                echo "$unformatted"
                exit 1
            fi
            echo "gofmt: clean"
            go vet ./...
            echo "go vet: clean"
        ' || return 1
        in_docker "$LINT_IMAGE" '
            golangci-lint run --timeout 3m ./...
            echo "golangci-lint: clean"
        ' || return 1
        ;;

    build)
        # 1) a static binary, kept as a pipeline artifact
        in_docker "$GO_IMAGE" "
            go build -trimpath \
                -ldflags '-s -w -X example.com/ci-demo/internal/httpapi.Version=$APP_TAG' \
                -o /artifacts/api ./cmd/api
            ls -lh /artifacts/api
        " || return 1
        # 2) the deployable image, built through the docker socket
        in_docker_dind "$DOCKER_IMAGE" "
            docker build -t $APP_IMAGE:$APP_TAG -t $APP_IMAGE:latest --build-arg VERSION=$APP_TAG /src
            docker image inspect $APP_IMAGE:$APP_TAG --format 'image {{.RepoTags}} size={{.Size}} bytes'
        " || return 1
        # dotenv artifact: tells the later jobs which image this pipeline produced
        # (the equivalent of GitLab's `artifacts:reports:dotenv`)
        printf 'APP_TAG=%s\n' "$APP_TAG" > "$ARTIFACTS_DIR/build.env"
        echo "artifacts: api, build.env (APP_TAG=$APP_TAG)"
        ;;

    test)
        # the race detector needs cgo, hence the toolchain install on alpine
        in_docker "$GO_IMAGE" '
            apk add --no-cache gcc musl-dev >/dev/null
            CGO_ENABLED=1 go test -race -count=1 -covermode=atomic \
                -coverprofile=/artifacts/coverage.out ./...
            echo "--- coverage ---"
            go tool cover -func=/artifacts/coverage.out | tail -4
            go tool cover -html=/artifacts/coverage.out -o /artifacts/coverage.html
        ' || return 1
        ;;

    deploy)
        # pick up the image tag produced by the build job (dotenv artifact)
        if [[ -f "$ARTIFACTS_DIR/build.env" ]]; then
            # shellcheck source=/dev/null
            source "$ARTIFACTS_DIR/build.env"
            echo "[deploy] deploying $APP_IMAGE:$APP_TAG (from build.env)"
        else
            echo "[deploy] no build artifact found — run the build job first" >&2
            return 1
        fi

        # mock deployment: start the freshly built image on an isolated network,
        # smoke-test it from a second container, then tear everything down.
        in_docker_dind "$DOCKER_IMAGE" "
            cleanup() {
                docker rm -f $DEPLOY_CONTAINER >/dev/null 2>&1 || true
                docker network rm $DEPLOY_NET  >/dev/null 2>&1 || true
            }
            trap cleanup EXIT

            cleanup
            docker network create $DEPLOY_NET >/dev/null
            docker run -d --name $DEPLOY_CONTAINER --network $DEPLOY_NET $APP_IMAGE:$APP_TAG >/dev/null
            echo '[deploy] container started, waiting for readiness...'

            ready=0
            i=1
            while [ \$i -le 20 ]; do
                if docker run --rm --network $DEPLOY_NET $SMOKE_IMAGE \
                       wget -qO- http://$DEPLOY_CONTAINER:8080/health >/tmp/health 2>/dev/null; then
                    ready=1; break
                fi
                i=\$((i+1)); sleep 0.5
            done
            if [ \$ready -ne 1 ]; then
                echo '[deploy] service never became healthy — rolling back'
                docker logs $DEPLOY_CONTAINER 2>&1 | tail -5
                exit 1
            fi

            echo '[deploy] GET /health      ->' \$(cat /tmp/health)
            echo '[deploy] POST /v1/analyze ->' \$(docker run --rm --network $DEPLOY_NET $SMOKE_IMAGE \
                    wget -qO- --header='Content-Type: application/json' \
                    --post-data='{\"text\":\"deploy smoke test deploy\",\"top_n\":2}' \
                    http://$DEPLOY_CONTAINER:8080/v1/analyze)

            echo '[deploy] smoke tests passed, releasing the mock environment'
        " || return 1
        ;;

    *)
        echo "unknown stage: $stage" >&2
        return 2
        ;;
    esac
}

# --------------------------------------------------------------- pipeline ---
is_allowed_failure() {
    local job="$1"
    for a in ${ALLOW_FAILURE[@]+"${ALLOW_FAILURE[@]}"}; do
        [[ "$a" == "$job" ]] && return 0
    done
    return 1
}

main() {
    local selected=("$@")
    [[ ${#selected[@]} -eq 0 ]] && selected=("${STAGES[@]}")

    mkdir -p "$ARTIFACTS_DIR" "$LOG_DIR"
    docker volume create "$MOD_CACHE_VOL"   >/dev/null
    docker volume create "$BUILD_CACHE_VOL" >/dev/null

    log "${C_BOLD}pipeline${C_RESET}  workspace=$WORKSPACE  tag=$APP_TAG"
    log "${C_DIM}stages: ${selected[*]}${C_RESET}"

    local -a names=() results=() times=()
    local failed=0 started total_start
    total_start=$(date +%s)

    for job in "${selected[@]}"; do
        if (( failed )); then
            section "$job (skipped — an earlier job failed)"
            names+=("$job"); results+=("skipped"); times+=("0")
            continue
        fi

        section "$job"
        started=$(date +%s)
        if runjob "$job" 2>&1 | tee "$LOG_DIR/$job.log"; then
            local elapsed=$(( $(date +%s) - started ))
            ok "$job (${elapsed}s)  log: .ci/logs/$job.log"
            names+=("$job"); results+=("passed"); times+=("$elapsed")
        else
            local elapsed=$(( $(date +%s) - started ))
            if is_allowed_failure "$job"; then
                warn "$job failed but is allow_failure (${elapsed}s)"
                names+=("$job"); results+=("warned"); times+=("$elapsed")
            else
                bad "$job (${elapsed}s)  log: .ci/logs/$job.log"
                names+=("$job"); results+=("failed"); times+=("$elapsed")
                failed=1
            fi
        fi
    done

    printf '\n%s%s%s\n' "$C_BOLD" "pipeline summary" "$C_RESET"
    printf '%s\n' "--------------------------------------"
    local i
    for i in "${!names[@]}"; do
        case "${results[$i]}" in
            passed)  printf '  %sok%s      %-10s %3ss\n' "$C_GREEN"  "$C_RESET" "${names[$i]}" "${times[$i]}" ;;
            failed)  printf '  %sfailed%s  %-10s %3ss\n' "$C_RED"    "$C_RESET" "${names[$i]}" "${times[$i]}" ;;
            warned)  printf '  %swarn%s    %-10s %3ss\n' "$C_YELLOW" "$C_RESET" "${names[$i]}" "${times[$i]}" ;;
            skipped) printf '  %sskipped%s %-10s   -\n'  "$C_DIM"    "$C_RESET" "${names[$i]}" ;;
        esac
    done
    printf '%s\n' "--------------------------------------"
    printf '  total: %ss\n' "$(( $(date +%s) - total_start ))"

    if (( failed )); then
        printf '\n%sPIPELINE FAILED%s\n' "$C_RED$C_BOLD" "$C_RESET"
        return 1
    fi
    printf '\n%sPIPELINE PASSED%s\n' "$C_GREEN$C_BOLD" "$C_RESET"
    return 0
}

if [[ "${1:-}" == "--list" ]]; then
    printf '%s\n' "${STAGES[@]}"
    exit 0
fi

main "$@"
