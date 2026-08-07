@RTK.md

# Etpeto agent guide

Etpeto is an open source application for structured Brazilian public-procurement planning documents.

## Sources of truth

- Product and architecture: `docs/specs/2026-08-06-etpeto-design.md`
- Active implementation plan: `docs/plans/2026-08-06-foundation-walking-skeleton.md`
- Canonical commands: `Makefile`

## Required workflow

- Use test-driven development for every behavior change: red, green, refactor.
- Keep the document model independent from Lexical and Typst.
- Keep API, renderer, and frontend contracts explicit and versioned.
- Run the smallest relevant test during development and `make check` before completion.
- Update specs or architecture docs when a public contract changes.
- Never commit credentials, tokens, personal data, document contents, or generated local artifacts.
- Do not add infrastructure or dependencies that are outside the approved design without an ADR.

More specific `AGENTS.md` files may add rules inside their directories. The nearest instruction file wins when rules differ.

