# Owned 1:N entities are child resources, not inline blocks

Entities that have an **independent CRUD endpoint** and an **independent id** but no independent
identity outside a parent — e.g. host interfaces via `hostinterface.*`, items via `item.*` — are
modeled as **standalone child resources** with a required parent reference (`host_id`,
`template_id`), never as inline blocks on the parent. This complements
[[0003-asymmetric-attachment-model]] (which covered M:N) with a third pattern for owned 1:N.

## Rationale

- **Independent lifecycle.** Adding, removing, or replacing a child doesn't churn the parent
  resource. The Zabbix API supports this directly via the child object's own CRUD methods.
- **Clean cross-resource references.** A `zabbix_item` (future) pointing at a specific interface
  is `interface_id = zabbix_host_interface.snmp.id`, not an inline-list-index lookup.
- **Composability.** Different modules can manage the parent's body and its children
  independently without fighting.
- **Consistency.** Matches the AWS pattern (`aws_network_interface`, `aws_route`,
  `aws_s3_object`) and keeps `zabbix_host_interface` shaped the same way `zabbix_item` will be.

## Considered Options

- **Inline blocks on the parent** — rejected: forces host-resource churn on every interface
  edit and would create an inconsistency with the future `zabbix_item` resource.

## Consequences

- **Cross-child constraints cannot be validated at plan time.** Example: "at most one
  `zabbix_host_interface` per type may be `main = true`" surfaces as a Zabbix apply-time error,
  not a plan error. Documented loudly; the provider cannot span resources at validation.
- **Implicit ordering footgun.** Templates whose items need a specific interface (e.g. agent)
  require explicit dependency wiring (`depends_on` or a real reference). Documented; not
  preventable from the link resource alone.
