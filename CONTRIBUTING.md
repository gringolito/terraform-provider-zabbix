# Contributing to terraform-provider-zabbix

Thank you for taking the time to contribute! All contributions are welcome: bug reports, feature
requests, documentation improvements, and code changes.

## Code of Conduct

This project follows the
[Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
By participating, you agree to uphold it. Report unacceptable behavior to the project maintainer.

## Prerequisites

Before you begin, make sure you have the following installed:

| Tool | Version |
| ---- | ------- |
| [Go](https://go.dev/dl/) | See `go.mod` |
| [Docker](https://docs.docker.com/get-docker/) + [Docker Compose](https://docs.docker.com/compose/install/) | Any recent version |
| [Terraform](https://developer.hashicorp.com/terraform/downloads) | >= 1.13 |
| GNU Make | Any recent version |

## Development Setup

1. **Fork** the repository and clone your fork:

   ```shell
   git clone https://github.com/<your-username>/terraform-provider-zabbix.git
   cd terraform-provider-zabbix
   ```

2. **Download dependencies:**

   ```shell
   go mod download
   ```

3. **Verify the build:**

   ```shell
   make build
   ```

## Make Workflow

Before opening a pull request, run through the full make workflow in order:

```shell
make fmt        # Apply Go and Terraform formatting
make generate   # Regenerate auto-generated docs and files
make build      # Compile the provider binary
make lint       # Run linters (golangci-lint + terraform fmt check)
make acc-tests  # Run unit tests and acceptance tests against a live Zabbix instance
```

> **Note:** `make acc-tests` automatically starts and tears down a Zabbix Docker stack. Never
> invoke acceptance tests with `TF_ACC=1 go test ./...` directly, always use `make acc-tests`.

## Running Tests

Acceptance tests require a live Zabbix instance, which `make acc-tests` manages automatically:

```shell
# Start Zabbix and run all tests (recommended)
make acc-tests

# For iterative debugging: manage the stack manually
make testacc-up    # Start Zabbix
make testacc       # Run tests (requires TF_ACC=1, set automatically)
make testacc-down  # Tear down Zabbix
```

Tests run against Zabbix **7.0 LTS** in CI. If your change affects 7.2 or 7.4 support, verify
manually against those versions.

## Documentation

Documentation under `docs/` is auto-generated from two sources:

- **Schema descriptions** — `MarkdownDescription` fields on each resource/data source attribute.
- **Examples** — `.tf` files under `examples/resources/<name>/resource.tf` and
  `examples/data-sources/<name>/data-source.tf`.

When adding or changing a resource or data source:

1. Update the corresponding example file under `examples/`.
2. Run `make generate` to regenerate `docs/resources/*.md` and `docs/data-sources/*.md`.
3. Never edit files under `docs/` directly, they will be overwritten on the next `make generate`.

## Commit Conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Every commit
message must follow the format:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`,
`chore`, `revert`

All commits must be **signed and signed-off**:

```shell
git commit -S -s -m "feat(host): add support for SNMP v3 authentication"
```

Branch names follow the same convention: `<type>/<short-description>` in kebab-case.

Examples: `feat/snmp-v3-auth`, `fix/host-group-import`, `docs/update-contributing`

## Pull Request Guidelines

1. Run the full make workflow (`fmt` → `generate` → `build` → `lint` → `acc-tests`) before
   opening a PR.
2. Keep PRs focused, one logical change per PR.
3. Write or update tests for any behavioral change.
4. Update example files and run `make generate` for any schema change.
5. Use a Conventional Commits message as the PR title.

## Reporting Issues

Use [GitHub Issues](https://github.com/gringolito/terraform-provider-zabbix/issues) to report
bugs or request features. Search existing issues before opening a new one.

For bug reports, include:

- Provider version
- Terraform version
- Zabbix version
- Minimal reproducing `.tf` configuration
- Actual vs. expected behavior
