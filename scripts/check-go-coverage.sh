#!/bin/sh
set -eu

profile=${1:-coverage/go.out}
minimum=${GO_COVERAGE_MIN:-55}

if ! test -s "$profile"; then
  echo "coverage profile not found: $profile" >&2
  exit 1
fi

# Generated queries and process entry points stay out of the percentage, as the
# specification requires. Their behaviour is covered by integration tests and by
# the Compose smoke instead, not by unit statements.
filtered="${profile}.measured"
awk 'NR == 1 || ($0 !~ /\/database\/dbgen\// && $0 !~ /\/cmd\//)' "$profile" > "$filtered"

actual=$(go tool cover -func="$filtered" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')

if test -z "$actual"; then
  echo "could not read total coverage from $profile" >&2
  exit 1
fi

if ! awk -v actual="$actual" -v minimum="$minimum" 'BEGIN {exit !(actual + 0 >= minimum + 0)}'; then
  echo "Go coverage ${actual}% is below the ${minimum}% floor" >&2
  exit 1
fi

echo "Go coverage ${actual}% meets the ${minimum}% floor (generated code and entry points excluded)"
