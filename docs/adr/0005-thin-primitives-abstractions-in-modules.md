# Thin provider primitives; opinionated abstractions live in modules

The provider exposes faithful, roughly 1:1 mappings of Zabbix API objects (e.g. `zabbix_action`
carries conditions, escalation operation steps, and recovery/update operations directly) rather
than synthesizing higher-level resources such as a `zabbix_notification_policy`. Opinionated,
ergonomic bundles are composed as **Terraform modules** built from these primitives. This matches
the AWS-inspired model already chosen ([[0003-asymmetric-attachment-model]]), keeps drift mapping
honest, avoids hiding API capability behind leaky abstractions, and follows idiomatic Terraform:
providers expose primitives, modules compose them.

## Consequences

- "Alert/notification policy" is a module, not a resource — it maps onto a faithful `zabbix_action`.
- The same bar applies to all future requests for composite/convenience resources.
- End users write more HCL in exchange for predictability, clean drift, and full API coverage.
- Example modules may ship in-repo to demonstrate ergonomic composition.

## Exception: discriminated-union API objects

A single Zabbix API endpoint that uses a **fixed type discriminator** (a `type` field that is
immutable after creation, with non-overlapping per-type attribute sets) MAY be exposed as
separate resources per type rather than a single unified resource. This is a deliberate exception
to the 1:1 rule, justified when all three conditions hold:

1. The discriminator is **immutable** — the type cannot be changed after create (a `destroy`
   and re-create is required to change type).
2. The per-type attribute sets are **non-overlapping** — no attribute appears in more than one
   type's schema.
3. A unified resource would require a **plan-time validator** to enforce a constraint the schema
   itself cannot express (exactly one of N mutually exclusive blocks must be set).

Each type-specific resource must still map faithfully to its slice of the API without hiding
fields or synthesising behaviour. See [[0009-type-specific-media-type-resources]] for the
reference application of this exception.
