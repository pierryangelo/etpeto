#!/bin/sh
# Exercises the main journey against a running stack: sign in, create an ETP,
# save a revision, render the official PDF, prove that a stale save conflicts,
# and sign out. It uses only the public API, exactly as the browser does.
set -eu

base_url="${ETPETO_BASE_URL:-http://localhost:8080}"
email="${ETPETO_ADMIN_EMAIL:?set ETPETO_ADMIN_EMAIL}"
password="${ETPETO_ADMIN_PASSWORD:?set ETPETO_ADMIN_PASSWORD}"

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT
cookies="$work_dir/cookies.txt"

fail() {
  echo "smoke-journey: $1" >&2
  exit 1
}

status_of() {
  # status_of METHOD PATH [DATA_FILE]
  if [ "$#" -eq 3 ]; then
    curl --output /dev/null --silent --write-out '%{http_code}' \
      --request "$1" "$base_url$2" \
      --header 'Content-Type: application/json' \
      --header "X-Etpeto-Csrf: $csrf" \
      --cookie "$cookies" --data @"$3"
  else
    curl --output /dev/null --silent --write-out '%{http_code}' \
      --request "$1" "$base_url$2" \
      --header "X-Etpeto-Csrf: $csrf" \
      --cookie "$cookies"
  fi
}

echo "1/6 signing in"
printf '{"email":"%s","password":"%s"}' "$email" "$password" > "$work_dir/login.json"
curl --fail --silent --show-error --output /dev/null \
  --request POST "$base_url/api/v1/auth/local/login" \
  --header 'Content-Type: application/json' \
  --cookie-jar "$cookies" --data @"$work_dir/login.json" ||
  fail "login failed"
csrf=$(awk '/__Host-etpeto_csrf/ {print $7}' "$cookies")
[ -n "$csrf" ] || fail "login did not publish a CSRF token"

echo "2/6 creating a document"
printf '{"title":"ETP de verificação automatizada"}' > "$work_dir/create.json"
curl --fail --silent --show-error --output "$work_dir/created.json" \
  --request POST "$base_url/api/v1/documents" \
  --header 'Content-Type: application/json' \
  --header "X-Etpeto-Csrf: $csrf" \
  --cookie "$cookies" --data @"$work_dir/create.json" ||
  fail "document creation failed"

document_id=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["id"])' \
  "$work_dir/created.json")
[ -n "$document_id" ] || fail "created document has no identifier"

echo "3/6 saving a revision"
python3 - "$work_dir/created.json" "$work_dir/save.json" <<'PY'
import json, sys

stored = json.load(open(sys.argv[1]))
content = stored["content"]
content["sections"][0]["elements"][0]["richText"] = {
    "root": {
        "type": "root",
        "version": 1,
        "children": [
            {
                "type": "paragraph",
                "version": 1,
                "children": [
                    {
                        "type": "text",
                        "version": 1,
                        "text": "Necessidade registrada pela verificação automatizada.",
                    }
                ],
            }
        ],
    }
}
json.dump({"expectedRevision": stored["revision"], "content": content}, open(sys.argv[2], "w"))
PY
saved=$(status_of PUT "/api/v1/documents/$document_id" "$work_dir/save.json")
[ "$saved" = "200" ] || fail "saving returned $saved, expected 200"

echo "4/6 refusing a stale save"
conflict=$(status_of PUT "/api/v1/documents/$document_id" "$work_dir/save.json")
[ "$conflict" = "409" ] || fail "stale save returned $conflict, expected 409"

echo "5/6 rendering the official PDF"
curl --fail --silent --show-error --output "$work_dir/preview.pdf" \
  --request POST "$base_url/api/v1/documents/$document_id/preview" \
  --header "X-Etpeto-Csrf: $csrf" --cookie "$cookies" ||
  fail "preview failed"
head -c 5 "$work_dir/preview.pdf" | grep -q '%PDF' ||
  fail "preview response is not a PDF"

echo "6/6 signing out"
signed_out=$(status_of POST /api/v1/auth/logout)
[ "$signed_out" = "204" ] || fail "logout returned $signed_out, expected 204"

after_logout=$(curl --output /dev/null --silent --write-out '%{http_code}' \
  "$base_url/api/v1/documents" --cookie "$cookies")
[ "$after_logout" = "401" ] || fail "session survived logout with status $after_logout"

echo "smoke-journey: main journey passed"
