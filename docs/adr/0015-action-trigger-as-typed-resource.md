# Type-specific resource for trigger actions

The `action.*` Zabbix API endpoint uses `event_source` as an immutable type discriminator
with five distinct shapes: trigger (0), discovery (1), autoregistration (2), internal (3),
and service (4). Rather than a single `zabbix_action` resource, the provider exposes
per-source resources. The first is `zabbix_action_trigger` (`event_source = 0`).

## Rationale

`event_source` is fixed at creation time — changing it requires destroy-and-recreate.
Per-source condition types and operation types are non-overlapping (e.g. discovery actions
may add hosts; trigger actions may not). A unified resource would require a `ConfigValidator`
to enforce per-source constraints the schema cannot express natively. This satisfies all three
conditions of the discriminated-union exception in [[0005-thin-primitives-abstractions-in-modules]].
Applied the same way as [[0009-type-specific-media-type-resources]].

## Considered Options

- **Single `zabbix_action` hardcoded to trigger** — rejected: wastes the `zabbix_action`
  name and prevents a clean unified data source later; hides the discriminator from users.
- **Single `zabbix_action` with `event_source` attribute** — rejected: `event_source` is
  immutable (ForceNew), per-source attributes diverge, requires plan-time validators.
- **Per-source typed resources (chosen)** — `zabbix_action_trigger` now; others added
  as they come into scope. Namespace reserved cleanly; schema is self-describing.

## Consequences

- v1 ships `zabbix_action_trigger` only. Discovery, autoregistration, internal, and service
  action resources are out of scope until a future milestone adds them.
- `event_source` is not a schema attribute — it is hardcoded in the resource implementation.
- Import requires knowing the action type (`terraform import zabbix_action_trigger.foo <id>`).
- The resource name carries the constraint so no plan-time validator is needed.
