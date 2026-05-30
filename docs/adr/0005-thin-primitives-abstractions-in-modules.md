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
