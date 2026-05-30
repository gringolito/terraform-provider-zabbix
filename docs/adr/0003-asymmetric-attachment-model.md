# Cardinality-driven resource model (inline vs attachment), not blind AWS mimicry

We adopt the AWS owner/consumer/attachment pattern, but split modeling by **cardinality**
rather than applying it uniformly. **Mandatory** many-to-many links are modeled as a required,
exclusive inline set on the consumer (`host_group_ids` on `zabbix_host`, `template_group_ids`
on `zabbix_template`) because the API forbids creating the consumer without them — a pure
attachment resource has a create-time chicken-and-egg problem. **Optional** many-to-many links
are modeled as standalone [[#Attachment resource|attachment resources]] (`zabbix_host_template_link`,
backed by `host.massadd`/`massremove`), and the same relationship is never *also* exposed as an
inline list (mixing causes perpetual diffs — the AWS security-group-rule footgun).

## Considered Options

- **Pure AWS symmetry** (all links as attachment resources) — rejected: breaks host/template
  create, which requires ≥1 group, forcing a "primary group" bootstrap hack.
- **Pure inline** (groups and templates both inline) — rejected: loses the decoupling that lets
  multiple configs manage optional template links independently.

## Consequences

- Initial resource set: `zabbix_host_group`, `zabbix_template_group`, `zabbix_template`,
  `zabbix_host`, `zabbix_host_template_link` (attachment), plus a data source per resource.
- Inline group sets are authoritative: out-of-band group changes are corrected on apply.
