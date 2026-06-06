# Project conventions

## Git conventions

All commits must be signed and signed-off. Use `git commit -S -s` or configure `commit.gpgsign = true` and always pass `-s`.

## PR checklist

Before opening any PR that is not documentation-only, run:

1. `make generate` — regenerate any auto-generated files
2. `make build` — ensure the project compiles
3. `make lint` — ensure there are no lint errors
4. `make acc-tests` — run both unit tests and acceptance tests

## Docs generation

`make generate` uses `tfplugindocs` to auto-generate `docs/resources/*.md` and
`docs/data-sources/*.md` from two sources:

- **Schema descriptions** — the `MarkdownDescription` fields on each resource/data source schema attribute.
- **Examples** — `.tf` files under `examples/resources/<resource_name>/resource.tf` and
  `examples/data-sources/<data_source_name>/data-source.tf`.

When adding or changing a resource or data source, always update the corresponding example file
under `examples/` before running `make generate`. Never edit the generated files under `docs/`
directly — they will be overwritten.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the default five-role vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — one `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
