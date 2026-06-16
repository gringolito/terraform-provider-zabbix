# Singleton resources use adopt-on-create, reset-on-delete lifecycle

Some Zabbix objects are global singletons — they always exist, cannot be created or deleted via
the API, and are managed exclusively through an `*.update` call. `zabbix_authentication` is the
first such resource in this provider.

The chosen lifecycle is:

- **Create** — reads the current singleton state, then calls `authentication.update` with the
  desired configuration. Does not error if the object already has non-default values; adoption
  of pre-existing state is intentional.
- **Read** — calls `authentication.get` and maps the result into Terraform state as usual.
- **Update** — calls `authentication.update` with the desired configuration.
- **Delete** — calls `authentication.update` with hardcoded documented defaults, effectively
  resetting the singleton. Emits a `resp.Diagnostics.AddWarning` (visible in `terraform destroy`
  output) and a `tflog.Warn` explaining that the resource cannot be truly deleted.

The resource carries a synthesized constant `id` of `"authentication"` — there is no Zabbix API
ID for this object. `ImportState` uses `resource.ImportStatePassthroughID`; the user passes the
literal string `"authentication"` as the import ID.

## Considered Options

- **Create errors if object already has non-default values** — rejected: this is a singleton
  that is always provisioned by Zabbix. Erroring on adopt would make the resource unusable in
  any environment where the authentication config has ever been touched. The adoption pattern
  follows established Terraform community practice for singleton/global resources.
- **Delete is a no-op** — rejected: leaving Terraform's `terraform destroy` silently doing
  nothing violates user expectations. Resetting to defaults is a meaningful action and clearly
  documented; a no-op would be invisible and surprising.
- **Delete truly deletes the object** — rejected: the Zabbix API has no `authentication.delete`
  endpoint. The object cannot be removed.

## Consequences

- The singleton helper lives in `internal/provider/singleton.go` as unexported functions in the
  `provider` package, consistent with other shared provider-layer helpers (`tags.go`,
  `media_type_common.go`). It can be extracted to its own package if a second singleton arrives.
- The hardcoded defaults in Delete must be sourced from the Zabbix 7.0 API documentation and
  annotated with a comment citing the reference, so future maintainers can verify them against
  a new Zabbix version.
- Users must be warned prominently in `MarkdownDescription` that declaring
  `zabbix_authentication` twice in the same configuration is a footgun — both blocks will fight
  over the same singleton and produce non-deterministic applies.
