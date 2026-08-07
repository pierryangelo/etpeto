#!/bin/sh
set -eu

for path in \
  AGENTS.md \
  RTK.md \
  Makefile \
  LICENSE \
  README.md \
  TODO.md \
  CONTRIBUTING.md \
  CODE_OF_CONDUCT.md \
  SECURITY.md \
  GOVERNANCE.md \
  SUPPORT.md \
  .github/workflows/ci.yml \
  .github/dependabot.yml \
  .github/CODEOWNERS \
  .github/pull_request_template.md \
  docs/adr/README.md \
  docs/architecture/foundation.md \
  docs/runbooks/development.md \
  docs/specs/2026-08-06-etpeto-design.md
do
  if ! test -s "$path"; then
    echo "missing required file: $path" >&2
    exit 1
  fi
done

grep -q '^help:' Makefile
grep -q 'GNU AFFERO GENERAL PUBLIC LICENSE' LICENSE
grep -q 'cp .env.example .env' README.md
grep -q '\[Arquitetura\]' README.md
grep -q '\[Pendências e roadmap\]' README.md
grep -q '\[Segurança\]' README.md
grep -q '\[Licença\]' README.md
