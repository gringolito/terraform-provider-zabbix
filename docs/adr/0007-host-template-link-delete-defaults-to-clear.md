# `zabbix_host_template_link` delete defaults to clear

On destroy, the link maps to Zabbix's `templates_clear` (delete inherited items, triggers,
graphs from the host) rather than bare `host.massremove templateids` (unlink only, leaving the
inherited entities as host-local orphans with `templateid=0`). This matches Terraform's
"destroy removes what this resource created" mental model. The behavior is configurable via
`on_destroy = "clear" | "unlink"` (default `"clear"`) so operators can opt out for
history-sensitive migrations (preserve `itemid` continuity across re-linking) or any case
where preserving inherited entities is intentional.

## Considered Options

- **Default `"unlink"`** — rejected: silently leaks host-local orphans across every link/unlink
  cycle, producing long-tail "I unlinked it but the items are still showing up" reports.
- **No attribute (hard-coded `"clear"`)** — rejected: removes the legitimate use case of
  preserving items during template migrations.

## Consequences

- Zabbix may refuse to clear inherited items that have host-local dependencies (e.g. a host-local
  trigger referencing an inherited item). We **surface the Zabbix error verbatim** — we do not
  cascade-delete the dependents.
- `on_destroy` is destroy-time-only; changing it on an existing resource has no effect until
  destroy.
