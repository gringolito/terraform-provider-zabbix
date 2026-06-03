# Type-specific resources for the media type discriminated union

The `mediatype.*` Zabbix API endpoint uses a `type` integer discriminator to represent four
distinct, non-overlapping shapes: email, SMS, script, and webhook. The type is fixed at creation
time — changing it requires destroying and re-creating the object. Rather than exposing a single
`zabbix_media_type` resource with four mutually exclusive optional settings blocks, the provider
exposes four type-specific resources: `zabbix_media_type_email`, `zabbix_media_type_sms`,
`zabbix_media_type_script`, and `zabbix_media_type_webhook`. This applies the discriminated-union
exception in [[0005-thin-primitives-abstractions-in-modules]].

## Rationale

A unified `zabbix_media_type` resource would have four optional settings blocks
(`email_settings`, `sms_settings`, `script_settings`, `webhook_settings`) where exactly one
must be set and must match the `type` attribute. The schema cannot express this constraint
natively — it requires a plan-time `ConfigValidator` to enforce it. The resulting resource is
self-contradicting: it presents four valid-looking configurations when only one is ever valid.

Type-specific resources make the schema self-describing. `zabbix_media_type_email` has email
settings as top-level attributes; there is no `type` attribute (it is hardcoded in the resource
implementation), no validator, and no spurious optional blocks.

## Considered Options

- **Single `zabbix_media_type` with plan-time validator** — rejected: requires a
  `ConfigValidator` to enforce a constraint the schema should express itself; presents four
  mutually exclusive blocks to users and tooling (IDE completion, `terraform validate`).
- **Four type-specific resources (chosen)** — each resource exposes only the attributes
  relevant to its type; no validator needed; HCL is self-describing.

## Consequences

- **Four resources, four data sources.** No unified `zabbix_media_type` data source ships by
  default; one may be added if a use case emerges that requires looking up a media type without
  knowing its type in advance.
- **Shared code lives in `media_type_common.go`.** A `MediaTypeBaseModel` embedded struct and a
  `commonMediaTypeSchemaAttributes()` builder hold the seven attributes shared across all types
  (`name`, `status`, `description`, `max_sessions`, `max_attempts`, `attempt_interval`,
  `message_templates`). This is the first shared-schema pattern in the provider.
- **`type` is not a schema attribute.** The type is encoded in the resource address
  (`zabbix_media_type_email.foo`) and hardcoded in each resource's model-to-API converter.
  Exposing it as a computed attribute would add noise without value.
- **Import requires knowing the type.** `terraform import zabbix_media_type_email.foo <id>`
  requires the operator to know the media type is email before importing. This is acceptable:
  the type is always visible in the Zabbix UI and is fixed for the lifetime of the object.
- **Deliberate departure from strict 1:1 API mapping** per [[0005-thin-primitives-abstractions-in-modules]].
  The rationale is schema clarity, not abstraction: each resource still maps faithfully to its
  slice of the API without hiding fields or synthesising behaviour.
