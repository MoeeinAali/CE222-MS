# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.23

FROM golang:${GO_VERSION}-alpine AS build

ARG VERSION=dev
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
        -ldflags "-s -w -X example.com/ci-demo/internal/httpapi.Version=${VERSION}" \
        -o /out/api ./cmd/api

FROM alpine:3.20 AS runtime

RUN addgroup -S api && adduser -S -G api -u 10001 api

COPY --from=build /out/api /usr/local/bin/api

USER api
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=2s --retries=3 \
    CMD wget --spider -q http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/api"]
