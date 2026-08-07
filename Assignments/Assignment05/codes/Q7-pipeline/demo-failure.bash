#!/usr/bin/env bash
#
# demo-failure.bash — proves the pipeline gates actually block.
#
# It copies the project into a scratch workspace, injects a *real* defect, and
# runs runner.bash against that copy, so the committed sources stay untouched.
#
#   ./demo-failure.bash lint     # unformatted code + an unchecked error
#   ./demo-failure.bash build    # Dockerfile references a missing file
#   ./demo-failure.bash test     # a unit test expectation is violated
#   ./demo-failure.bash deploy   # the deployed container never becomes healthy
#
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIND="${1:?usage: demo-failure.bash lint|build|test|deploy}"

SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/q7-fail-$KIND.XXXXXX")"
trap 'rm -rf "$SCRATCH"' EXIT

cp -R "$ROOT/app" "$SCRATCH/app"
rm -rf "$SCRATCH/app/.ci"

case "$KIND" in
lint)
    # badly formatted + an ignored error -> gofmt and errcheck both complain
    cat > "$SCRATCH/app/internal/textstats/broken.go" <<'GO'
package textstats

import "os"

func Dump( path string ) error {
        f,_ := os.Create(path)
    defer f.Close()
    f.WriteString("dump")
        return nil
}
GO
    ;;
build)
    # the image build references a file that does not exist
    printf '\nCOPY this-file-does-not-exist.txt /tmp/\n' >> "$SCRATCH/app/Dockerfile"
    ;;
test)
    # flip a real expectation: "the" occurs 3 times, claim 4
    sed -i.bak 's/Word: "the", Count: 3/Word: "the", Count: 4/' \
        "$SCRATCH/app/internal/textstats/textstats_test.go"
    rm -f "$SCRATCH/app/internal/textstats/textstats_test.go.bak"
    ;;
deploy)
    # the container starts but can never bind, so readiness never happens
    printf '\nENV LISTEN_ADDR=not-a-valid-address\n' >> "$SCRATCH/app/Dockerfile"
    ;;
*)
    echo "unknown demo: $KIND" >&2
    exit 2
    ;;
esac

echo "### failure demo: $KIND   (scratch workspace: $SCRATCH)"
echo

WORKSPACE="$SCRATCH/app" \
CI_DIR="$SCRATCH/.ci" \
APP_IMAGE="ci-demo-faildemo" \
    "$ROOT/runner.bash" || echo "(runner exited with a non-zero status, as expected)"
