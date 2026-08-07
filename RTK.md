# Terminal output conventions

- Prefer `rg` and `rg --files` for search.
- Prefer `rtk git status`, `rtk git diff`, `rtk git add`, and `rtk git commit` when `rtk` is available.
- Use ordinary Git when RTK would hide evidence needed for diagnosis or review.
- Keep commands non-interactive and scoped to the repository.
- Never expose secrets or document contents in terminal output.

