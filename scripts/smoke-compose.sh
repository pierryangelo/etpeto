#!/bin/sh
set -eu

base_url="${ETPETO_BASE_URL:-http://localhost:8080}"

curl --fail --silent --show-error "$base_url/api/v1/service-info" |
  grep -q '"name":"Etpeto"'
curl --fail --silent --show-error "$base_url/" |
  grep -q '<title>Etpeto</title>'

# A client-routed path must serve the application shell, otherwise every URL
# except the root breaks on reload.
curl --fail --silent --show-error "$base_url/documentos" |
  grep -q '<title>Etpeto</title>'

# The preview iframe holds the PDF as an object URL, which the policy must allow.
curl --fail --silent --show-error --head "$base_url/" |
  grep -qi "content-security-policy:.*frame-src 'self' blob:"

# The renderer stays unreachable without a session.
status=$(curl --output /dev/null --silent --write-out '%{http_code}' \
  --request POST \
  "$base_url/api/v1/documents/00000000-0000-0000-0000-000000000000/preview")
test "$status" = "401" || {
  echo "anonymous preview returned $status, expected 401" >&2
  exit 1
}
