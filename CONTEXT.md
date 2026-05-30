# Zabbix Terraform Provider

A Terraform provider for managing Zabbix configuration (hosts, groups, templates, triggers,
notifications, dashboards) declaratively via the Zabbix JSON-RPC API, built on the
terraform-plugin-framework.

## Language

### Version support

**Targeted**:
A Zabbix version that is exercised in our CI integration matrix and whose API schema we
commit to. Currently **7.0 LTS**.
_Avoid_: "supported" (ambiguous — split into Targeted vs Tolerated)

**Tolerated**:
A Zabbix version the provider will run against on a best-effort basis via the
version-tolerant client (detected at configure time, warns if untested), but whose
version-specific schema we make no guarantees about. Currently **7.2 and 7.4** (7.2 is
upstream-EOL, included on a strict best-effort basis only).
_Avoid_: "supported"

**Version-tolerant client**:
The Zabbix API client design that detects `apiinfo.version` at provider configure time,
sends only fields the provider manages, ignores unknown response fields, warns on
[[Tolerated]] versions, and fails fast on versions below the [[Targeted]] baseline.

### Core entities

**Host**:
A monitored entity. Belongs to **≥1** [[#Host group|host group]] (mandatory) and links to
**0..N** [[#Template|templates]] (optional). Has two names: a **technical name** (`host` field —
unique, used in trigger expressions) and a **visible name** (`name` field — display only, not
unique). Lookups key off the technical name or the id.

**Host group**:
A container that organizes [[#Host|hosts]] (`hostgroup.*` API). Distinct from a
[[#Template group|template group]].
_Avoid_: bare "group"

**Template**:
A reusable bundle of items, triggers, graphs, etc. that can be linked to [[#Host|hosts]].
Belongs to **≥1** [[#Template group|template group]] and may nest **0..N** other templates.

**Template group**:
A container that organizes [[#Template|templates]] (`templategroup.*` API). Split out from
[[#Host group|host groups]] in Zabbix 6.2; do not conflate the two.
_Avoid_: bare "group"

### Resource modeling

**Owner / Consumer / Attachment**:
The AWS-inspired pattern we use for many-to-many relationships. An _owner_ provides a
capability (host group, template, template group), a _consumer_ uses it (host, template),
and an _attachment_ is the link. How a relationship is modeled depends on cardinality (below).

**Exclusive inline membership**:
A **mandatory** many-to-many relationship is modeled as a required set attribute on the
consumer resource (e.g. `host_group_ids` on `zabbix_host`), owned exclusively by that
resource. Used when the API forbids creating the consumer without the link (a host cannot be
created without a host group).

**Attachment resource**:
An **optional** many-to-many relationship is modeled as a standalone resource (e.g.
`zabbix_host_template_link`) backed by `*.massadd`/`*.massremove`. The same relationship is
**never** also exposed as an inline list — mixing the two causes perpetual diffs.

**Owned child resource**:
An **owned 1:N** entity with an independent CRUD endpoint and id but no independent identity
outside its parent (e.g. `hostinterface.*`, `item.*`) is modeled as a standalone resource with a
required `<parent>_id` reference (`host_id`, `template_id`) — _never_ as an inline block on the
parent. Gives the child its own lifecycle, makes cross-resource references clean (an item
points at `zabbix_host_interface.snmp.id`), and matches the AWS pattern for sub-objects
(`aws_network_interface`, `aws_route`). Example: `zabbix_host_interface`.

### Template management

Two distinct ways to get a [[#Template|template]] into Zabbix, with different lifecycles:

**Ad-hoc template**:
A template authored declaratively via the `zabbix_template` resource (its shell: name,
template groups, macros, linked templates). Fully owned by Terraform state with real
field-level drift detection. The _owner_ in the [[#Owner / Consumer / Attachment]] model.

**Library template**:
A template loaded from an external export file (XML/YAML/JSON) via [[#Template import]].
Treated as long-lived **shared content**: not owned by Terraform state, and **deletion is a
deliberate human act**, never a `terraform destroy` side-effect (other hosts may depend on it).
Referenced elsewhere via the `zabbix_template` data source.
_Avoid_: "imported template" used to imply Terraform owns its lifecycle.

**Template import**:
The imperative `configuration.import` load, modeled as a **stateless Terraform Action**
(invokable standalone via `-invoke`, or via `after_create`/`after_update` lifecycle triggers).
Because actions have no state, it provides **no drift detection and no cleanup** — by design,
consistent with [[#Library template]] semantics.

### Alerting and notifications

**Trigger**:
A logical expression over item data defining a problem condition (`trigger.*`). Belongs to a
[[#Host|host]] or [[#Template|template]]; its expression references items that must already
exist (typically provided by a linked template — item _authoring_ is out of current scope).

**Action**:
The alerting rule (`action.*`): "when a [[#Trigger|trigger]] fires matching these conditions,
run these operations." Carries conditions, escalation operation steps, and recovery/update
operations. This _is_ the alert/notification policy — there is no separate "policy" object.
_Avoid_: "alert" (the rule) — say **Action**.

**Media type**:
A notification delivery channel — email, SMS, webhook, Slack, etc. (`mediatype.*`).

**User media**:
A specific user's address (e.g. an email) on a [[#Media type|media type]]. The recipient end
of a notification.

## Flagged ambiguities

- **"group"** — always disambiguate to [[#Host group|host group]] or
  [[#Template group|template group]]. They are separate API objects in 7.0.
- **"host name"** — disambiguate to **technical name** (`host`, unique) or **visible name**
  (`name`, not unique). Identity and lookups use the technical name or id.
- **"alert"** — never a writable object. The _rule_ is an [[#Action]]; the _sent message_ is
  runtime-only, exposed at most as a read-only data source.
- **"alert policy" / "notification policy"** — not a native Zabbix object. It maps to an
  [[#Action]]'s conditions + escalation steps. Ergonomic composition belongs in a Terraform
  module, not a provider resource.

## Example dialogue

> **Dev:** Can I just create a host with no groups and attach everything later?
> **Domain expert:** No — a host needs at least one host group at create time, so
> `host_group_ids` is required and exclusive on `zabbix_host`. Templates are different:
> they're optional, so you create the host first and link templates with separate
> `zabbix_host_template_link` resources.
> **Dev:** Why not put a `template_ids` list on the host too, for convenience?
> **Domain expert:** Because then two places fight over the same linkage and you get
> perpetual diffs. One relationship, one place: attachment resource only.
