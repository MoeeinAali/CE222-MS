#!/usr/bin/env bash
#
# verify-nexus.sh — uses the repositories created by setup-nexus.sh.
#
#   A) build the Q4 project with its dependencies coming from pypi-proxy
#      (the very MIRROR_URL mechanism that question asks for)
#   B) show that npm-proxy answers as a registry
#   C) push an image to docker-hosted and pull it back
#
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-localhost:8082}"
ADMIN_USER="admin"
ADMIN_PASSWORD="${NEXUS_ADMIN_PASSWORD:-admin123}"

# inside a build container the host is reachable under this name
NEXUS_FROM_CONTAINER="${NEXUS_FROM_CONTAINER:-http://host.docker.internal:8081}"

Q4_DIR="$ROOT/../Q4-dockerize"
IMAGE_LOCAL="alpine:3.20"
IMAGE_REMOTE="$DOCKER_REGISTRY/hw5/alpine:3.20"

say() { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }

# --------------------------------------------- A) python deps via the proxy --
say "A) building the Q4 image with pip pointed at pypi-proxy"
echo "   index-url: $NEXUS_FROM_CONTAINER/repository/pypi-proxy/simple"
docker build \
    --no-cache-filter builder \
    --build-arg "MIRROR_URL=$NEXUS_FROM_CONTAINER/repository/pypi-proxy/simple" \
    --build-arg "MIRROR_TRUSTED_HOST=host.docker.internal:8081" \
    --target builder -t q4-app:from-nexus "$Q4_DIR" --progress plain 2>&1 \
  | grep -E "Looking in indexes|Successfully installed|Downloading" | head -8

say "packages now cached in the pypi-proxy blob store"
curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" \
     "$NEXUS_URL/service/rest/v1/components?repository=pypi-proxy" \
  | jq -r '.items[] | "\(.name) \(.version)"' | sort | head -12

# ---------------------------------------------------- B) npm proxy answers ---
say "B) npm-proxy metadata check"
curl -sS "$NEXUS_URL/repository/npm-proxy/react" | jq -r '"react latest = " + .["dist-tags"].latest'

# ------------------------------------------------- C) docker push and pull ---
say "C) docker login $DOCKER_REGISTRY"
echo "$ADMIN_PASSWORD" | docker login "$DOCKER_REGISTRY" -u "$ADMIN_USER" --password-stdin

say "tag + push"
docker pull -q "$IMAGE_LOCAL" >/dev/null
docker tag "$IMAGE_LOCAL" "$IMAGE_REMOTE"
docker push "$IMAGE_REMOTE"

say "remove the local copy, then pull it back from Nexus"
docker rmi "$IMAGE_REMOTE" >/dev/null
docker pull "$IMAGE_REMOTE"
docker run --rm "$IMAGE_REMOTE" cat /etc/alpine-release | sed 's/^/    alpine release: /'

say "component list of docker-hosted"
curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" \
     "$NEXUS_URL/service/rest/v1/components?repository=docker-hosted" \
  | jq -r '.items[] | "\(.name):\(.version)  (\(.format))"'

say "done"
