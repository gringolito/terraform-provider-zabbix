# zabbix_trigger resource carries no host_id or template_id

[[0008-owned-child-resources]] requires owned 1:N entities to carry a parent reference. We make an explicit exception for `zabbix_trigger`: the resource has no `host_id` or `template_id` attribute.

Two reasons. First, `trigger.create` accepts no parent ID — ownership is inferred by Zabbix from the expression itself, which embeds host/template references by technical name (e.g. `last(/web01/cpu.util)>90`). Adding a synthetic parent attribute would violate [[0005-thin-primitives-abstractions-in-modules]]. Second, a trigger expression can legitimately reference multiple hosts, making a single `host_id` attribute semantically incorrect.

Drift detection and CRUD reads use `trigger.get` filtered by `triggerid` — no parent scope is needed. Implicit Terraform dependency edges are created naturally when the expression interpolates `zabbix_host.web.host` or `zabbix_template.linux.host`.

## Considered Options

- **Add `host_id`/`template_id` as required attributes** — rejected: no such field exists in the `trigger.*` API; expression can span multiple hosts; violates the 1:1 thin-primitive rule.
- **Add as optional attributes for scoping only** — rejected: same API-fidelity problem; scoping for `trigger.get` is unnecessary since `triggerid` is sufficient; optional attributes with no write effect would be surprising.

## Consequences

- The `zabbix_trigger` data source (which looks up by description, not by id) DOES carry `host_id`/`template_id` as optional scope attributes — lookup by description is not globally unique and requires scoping.
- Users must wire dependencies explicitly via expression interpolation when the trigger depends on a managed host or template.
