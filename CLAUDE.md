# Project conventions

## Git conventions

All commits must be signed and signed-off. Use `git commit -S -s` or configure `commit.gpgsign = true` and always pass `-s`.

## PR checklist

Before opening any PR that is not documentation-only, run:

1. `make generate` — regenerate any auto-generated files
2. `make build` — ensure the project compiles
3. `make lint` — ensure there are no lint errors
4. `make acc-tests` — run both unit tests and acceptance tests

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the default five-role vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — one `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
