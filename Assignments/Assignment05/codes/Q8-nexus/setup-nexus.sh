#!/usr/bin/env bash
#
# setup-nexus.sh — provisions the Nexus instance started by docker-compose.yml.
#
# Everything the assignment asks for is done through the REST API so the whole
# setup is reproducible (and reviewable) instead of a sequence of UI clicks:
#
#   1. wait until Nexus is up and rotate the generated admin password
#   2. create a file blob store          -> hw5-blob
#   3. create proxy repositories on it   -> pypi-proxy, npm-proxy
#   4. create a group repository         -> pypi-group
#   5. enable the docker bearer-token realm
#   6. create a hosted docker registry   -> docker-hosted (connector :8082)
#
set -Eeuo pipefail

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_CONTAINER="${NEXUS_CONTAINER:-hw5-nexus}"
ADMIN_USER="admin"
ADMIN_PASSWORD="${NEXUS_ADMIN_PASSWORD:-admin123}"
BLOB_NAME="hw5-blob"

# an upstream mirror can be substituted here without touching anything else
PYPI_REMOTE="${PYPI_REMOTE:-https://pypi.org/}"
NPM_REMOTE="${NPM_REMOTE:-https://registry.npmjs.org}"

say()  { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }
note() { printf '    %s\n' "$*"; }

api() { # api <METHOD> <PATH> [json-body]
    local method="$1" path="$2" body="${3:-}"
    if [[ -n "$body" ]]; then
        curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" -X "$method" \
             -H 'Content-Type: application/json' -d "$body" \
             -w '\n%{http_code}' "$NEXUS_URL$path"
    else
        curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" -X "$method" \
             -w '\n%{http_code}' "$NEXUS_URL$path"
    fi
}

# create a resource, tolerating "already exists"
create() { # create <label> <PATH> <json-body>
    local label="$1" path="$2" body="$3" out code
    out="$(api POST "$path" "$body")"
    code="$(tail -1 <<<"$out")"
    case "$code" in
        201|204) note "created  $label" ;;
        400)     note "exists   $label (400 - already present)" ;;
        *)       note "FAILED   $label (http $code): $(sed '$d' <<<"$out")"; return 1 ;;
    esac
}

# ------------------------------------------------------------------- 1. up --
say "waiting for Nexus at $NEXUS_URL"
for i in $(seq 1 90); do
    if curl -sf "$NEXUS_URL/service/rest/v1/status" >/dev/null; then
        note "Nexus is available after ${i}0s (roughly)"
        break
    fi
    sleep 10
    [[ $i -eq 90 ]] && { echo "Nexus did not start in time" >&2; exit 1; }
done

say "admin credentials"
if curl -sf -u "$ADMIN_USER:$ADMIN_PASSWORD" "$NEXUS_URL/service/rest/v1/status/check" >/dev/null; then
    note "password already rotated to '$ADMIN_PASSWORD'"
else
    GENERATED="$(docker exec "$NEXUS_CONTAINER" cat /nexus-data/admin.password)"
    note "generated password: $GENERATED"
    curl -sS -u "$ADMIN_USER:$GENERATED" -X PUT \
         -H 'Content-Type: text/plain' -d "$ADMIN_PASSWORD" \
         "$NEXUS_URL/service/rest/v1/security/users/admin/change-password"
    note "rotated to: $ADMIN_PASSWORD"
fi

# ------------------------------------------------------------------ EULA ----
# Nexus Repository Community Edition (3.78+) answers 403 to every artifact
# download until the End User License Agreement is accepted. The onboarding
# wizard does this in the UI; here it is one REST call. Setting "accepted"
# back to false undoes it.
say "End User License Agreement"
if [[ "$(curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" "$NEXUS_URL/service/rest/v1/system/eula" | jq -r .accepted)" == "true" ]]; then
    note "already accepted"
else
    DISCLAIMER="$(curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" "$NEXUS_URL/service/rest/v1/system/eula" | jq -r .disclaimer)"
    note "accepting: https://links.sonatype.com/products/nxrm/ce-eula"
    code="$(jq -n --arg d "$DISCLAIMER" '{accepted:true, disclaimer:$d}' \
        | curl -sS -o /dev/null -w '%{http_code}' -u "$ADMIN_USER:$ADMIN_PASSWORD" \
               -X POST -H 'Content-Type: application/json' -d @- \
               "$NEXUS_URL/service/rest/v1/system/eula")"
    note "eula -> http $code"
fi

# ----------------------------------------------------------- 2. blob store --
say "blob store"
create "$BLOB_NAME" /service/rest/v1/blobstores/file \
    "{\"path\":\"$BLOB_NAME\",\"name\":\"$BLOB_NAME\"}"

# ------------------------------------------------------ 3. proxy repositories
say "proxy repositories (stored on $BLOB_NAME)"
create "pypi-proxy -> $PYPI_REMOTE" /service/rest/v1/repositories/pypi/proxy "$(cat <<JSON
{
  "name": "pypi-proxy",
  "online": true,
  "storage": { "blobStoreName": "$BLOB_NAME", "strictContentTypeValidation": true },
  "proxy":   { "remoteUrl": "$PYPI_REMOTE", "contentMaxAge": 1440, "metadataMaxAge": 1440 },
  "negativeCache": { "enabled": true, "timeToLive": 1440 },
  "httpClient": { "blocked": false, "autoBlock": true }
}
JSON
)"

create "npm-proxy -> $NPM_REMOTE" /service/rest/v1/repositories/npm/proxy "$(cat <<JSON
{
  "name": "npm-proxy",
  "online": true,
  "storage": { "blobStoreName": "$BLOB_NAME", "strictContentTypeValidation": true },
  "proxy":   { "remoteUrl": "$NPM_REMOTE", "contentMaxAge": 1440, "metadataMaxAge": 1440 },
  "negativeCache": { "enabled": true, "timeToLive": 1440 },
  "httpClient": { "blocked": false, "autoBlock": true }
}
JSON
)"

# ------------------------------------------------------- 4. group repository
say "group repository (one URL in front of several repositories)"
create "pypi-group [pypi-proxy]" /service/rest/v1/repositories/pypi/group "$(cat <<JSON
{
  "name": "pypi-group",
  "online": true,
  "storage": { "blobStoreName": "$BLOB_NAME", "strictContentTypeValidation": true },
  "group":   { "memberNames": ["pypi-proxy"] }
}
JSON
)"

# --------------------------------------------------------- 5. docker realm --
say "enabling the Docker Bearer Token realm"
# the ids come from GET /service/rest/v1/security/realms/available
REALMS='["NexusAuthenticatingRealm","DockerToken"]'
code="$(curl -sS -o /dev/null -w '%{http_code}' -u "$ADMIN_USER:$ADMIN_PASSWORD" \
        -X PUT -H 'Content-Type: application/json' -d "$REALMS" \
        "$NEXUS_URL/service/rest/v1/security/realms/active")"
note "realms/active -> http $code"

# -------------------------------------------------------- 6. docker hosted --
# forceBasicAuth=true makes the registry authenticate with plain HTTP Basic.
# With the bearer-token flow (forceBasicAuth=false) Nexus builds the token
# endpoint from its "Base URL" capability, which is unset on a fresh install
# and is not exposed by the REST API - `docker login` then fails with
# `unsupported protocol scheme "null"`.
say "hosted docker registry on port 8082"
create "docker-hosted" /service/rest/v1/repositories/docker/hosted "$(cat <<JSON
{
  "name": "docker-hosted",
  "online": true,
  "storage": { "blobStoreName": "$BLOB_NAME", "strictContentTypeValidation": true, "writePolicy": "ALLOW" },
  "cleanup": null,
  "docker":  { "v1Enabled": false, "forceBasicAuth": true, "httpPort": 8082 }
}
JSON
)"

# ------------------------------------------------------------------ report --
say "result"
printf '%-16s %-8s %s\n' NAME FORMAT URL
curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" "$NEXUS_URL/service/rest/v1/repositories" \
  | jq -r '.[] | [.name, .format, .url] | @tsv' \
  | awk -F'\t' '{printf "%-16s %-8s %s\n", $1, $2, $3}'

say "blob stores"
curl -sS -u "$ADMIN_USER:$ADMIN_PASSWORD" "$NEXUS_URL/service/rest/v1/blobstores" \
  | jq -r '.[] | "\(.name)\t(\(.type))"'
