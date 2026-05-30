# Build our own thin Zabbix JSON-RPC client

There is no official Zabbix Go SDK and no OpenAPI/OpenRPC schema (codegen isn't viable), and the
community libraries are coupled to the legacy SDKv2 provider lineage with uneven 7.0 coverage.
We build our own thin client instead: a generic transport (`Call(ctx, method, params) → raw JSON`,
handling the Bearer auth header, JSON-RPC envelope, error decoding, and `apiinfo.version`
detection) with per-resource typed structs layered on top. This is the only way to own the
[[version-tolerant client]] semantics (send only managed fields, ignore unknown response fields)
that a third-party library would fight.

## Consequences

- The transport sits behind an interface so the whole client is mockable; unit tests run with no network.
- Dependency injection is used where it makes sense (the client interface is injected into resources/data sources) to keep the unit-test seam clean.
- We own auth, transport, retry, and error mapping ourselves — more upfront code, but full control.
- Revisit if an official Zabbix Go client appears: the typed-struct layer is the migration cost.
