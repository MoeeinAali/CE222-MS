# Q7 — a GitLab-CI-like pipeline without a GitLab Runner

`runner.bash` plays the role of the runner: it owns the stage list, executes
every job **inside a container**, caches modules in named volumes, publishes
artifacts, fails fast and prints a pass/fail summary.

```bash
./runner.bash              # whole pipeline
./runner.bash lint test    # selected jobs
./runner.bash --list
./demo-failure.bash lint   # lint | build | test | deploy — proves the gates block
```

## Stages

| job | image | what it does | gate |
|---|---|---|---|
| `install` | `golang:1.23-alpine` | `go mod download` + `go mod verify` | checksum mismatch |
| `lint` | `golang:1.23-alpine`, `golangci-lint:v1.62.2` | `gofmt -l`, `go vet`, `golangci-lint` | any finding |
| `build` | `golang:1.23-alpine`, `docker:27-cli` | static binary → `.ci/artifacts/api`, then the deployable image | compile / image build error |
| `test` | `golang:1.23-alpine` | `go test -race -coverprofile` | any failing test |
| `deploy` | `docker:27-cli`, `alpine` | **mock**: run the image on a private network, smoke-test `/health` and `/v1/analyze`, tear down | service not healthy |

## Runner mechanics worth noting

* **docker executor** — each job is one `docker run --rm` with the workspace at
  `/src`; the two jobs that need the daemon get `/var/run/docker.sock` bound in.
* **cache** — `q7-go-mod-cache` and `q7-go-build-cache` volumes survive runs.
* **artifacts** — `.ci/artifacts/` (binary, `coverage.out`, `coverage.html`) and
  `.ci/logs/<job>.log` per job.
* **dotenv artifact** — `build` writes `build.env` with the image tag it
  produced and `deploy` sources it, mirroring GitLab `artifacts:reports:dotenv`.
* **fail-fast** — the first failing job marks every later job `skipped` and the
  script exits non-zero.

> Bash note: a failing command inside `runjob` cannot rely on `set -e`, because
> `set -e` is suspended while a function runs as an `if` condition. Every step
> therefore ends in an explicit `|| return 1`.

## The application under test

`app/` — a Go 1.23 HTTP service (`GET /health`, `POST /v1/analyze` returning
word statistics), 15 test cases, 70.5% statement coverage.
