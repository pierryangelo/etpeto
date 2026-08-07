#!/bin/sh
set -eu

base_url="${ETPETO_BASE_URL:-http://localhost:8080}"

curl --fail --silent --show-error "$base_url/api/v1/service-info" |
  grep -q '"name":"Etpeto"'
curl --fail --silent --show-error "$base_url/" |
  grep -q '<title>Etpeto</title>'
