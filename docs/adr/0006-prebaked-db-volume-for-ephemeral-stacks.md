---
status: proposed
---

# Pre-baked DB volume for fast, fully-ephemeral acceptance stacks

The first milestone runs acceptance tests against a **shared** docker-compose web+db stack, whose
boot is dominated by Postgres's one-time schema import — which is why the stack is shared per run
and isolation is logical (unique-prefixed objects + `CheckDestroy`). **Evaluate** baking a
pre-initialized Postgres data volume (or a custom image with the schema pre-loaded) so the stack
boots near-instantly and can be brought up/down **fully ephemerally** per run — or per parallel
version shard — removing the shared-stack constraint entirely.

## Trade-off to evaluate

- **Gain:** near-instant boot, fully ephemeral stacks, per-test/per-shard isolation becomes cheap.
- **Cost:** the baked artifact must be regenerated whenever the Zabbix schema changes — i.e. once
  per [[#Targeted|targeted]] version (7.0 now, 8.0 later) — adding a CI/build step and a staleness risk.

## Revisit when

Acceptance boot time becomes a CI bottleneck, or we want concurrent per-version ephemeral stacks.
