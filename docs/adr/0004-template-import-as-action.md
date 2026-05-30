# Template import is a stateless Action, not a resource

`configuration.import` is an imperative "apply this blob" verb, and we treat imported templates
as [[#Library template|library content]]: long-lived, shared, deleted only by deliberate human
action. So we model the import as a **stateless Terraform Action** (terraform-plugin-framework),
invokable standalone (`terraform apply -invoke=action.zabbix_template_import.x`) or wired to a
resource's `after_create`/`after_update` lifecycle. Referencing imported templates is done via
the `zabbix_template` data source; the ad-hoc `zabbix_template` resource is for templates whose
full lifecycle Terraform owns.

## Considered Options

- **Resource with content-hash drift + parsed-name delete** — rejected: a synthetic resource
  whose `Read` cannot truly reconcile against the server, and whose `destroy` could delete a
  template that hosts outside this workspace depend on.

## Consequences

- **No drift detection and no managed cleanup.** Actions have no state (a deliberate framework
  limitation today). Removing the action from config leaves imported templates in place — by design.
- Terraform 1.14 actions have no `destroy` lifecycle events, which reinforces (rather than blocks)
  the no-cleanup model.
- Re-import happens on manual `-invoke` or on every triggered apply; there is no "only when the
  source changed" path, because that would require state.
