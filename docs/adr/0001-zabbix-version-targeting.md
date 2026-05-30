# Target Zabbix 7.0 LTS, tolerate 7.2 and 7.4, plan for 8.0 LTS

We baseline the provider's API contract on **Zabbix 7.0 LTS** (full support through Jun 2027,
limited through Jun 2029). The client detects `apiinfo.version` at configure time, fails fast
below 7.0, and *tolerates* the standard releases **7.2 and 7.4** (warns, no schema guarantees).
7.2 is upstream-EOL but the version-tolerant client makes best-effort runtime support
essentially free, so we include it; 7.4's provider-relevant deltas are thin (OAuth SMTP media
types, new dashboard widgets — and dashboards are only a spike). 8.0 LTS is expected Q3 2026
and will be the next real target, so version detection is built to make adding it cheap.

## Consequences

- A single auth path: API token via `Authorization: Bearer` (6.4+ model), not the legacy `auth` body field.
- The client must be schema-tolerant (ignore unknown response fields, send only managed fields) so Tolerated versions don't break on additive API changes.
- CI integration tests run against 7.0 LTS now; 8.0 LTS joins the matrix when it ships. 7.2 and 7.4 are not part of the CI matrix — "tolerated" means runtime-tolerant, not CI-tested.
