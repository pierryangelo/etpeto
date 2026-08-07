#!/bin/sh
set -eu

for path in AGENTS.md RTK.md Makefile LICENSE docs/specs/2026-08-06-etpeto-design.md; do
  if ! test -s "$path"; then
    echo "missing required file: $path" >&2
    exit 1
  fi
done

grep -q '^help:' Makefile
grep -q 'GNU AFFERO GENERAL PUBLIC LICENSE' LICENSE
