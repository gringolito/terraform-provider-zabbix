# Model tags as a set of `{name, value}` objects, not a map

Wherever a Zabbix entity supports tags (hosts, templates, triggers, actions), the provider
schema exposes the `tags` attribute as a **`SetNestedAttribute` of `{name, value}` objects**,
not as a `map(string)`.

The `name` attribute is required. The `value` attribute is optional and defaults to `""`,
matching the Zabbix API's treatment of a missing value.

## Rationale

Zabbix allows multiple tags with the same name but different values on a single entity:

```
foo:bar
foo:baz
foo:qux
```

A `map(string)` can only hold one value per key, so it would silently discard all but one
`foo` tag. The `{name, value}` set preserves the full tag set without loss.

A set (rather than a list) is used because tag order is semantically meaningless and Zabbix
returns tags in arbitrary order. A list would produce spurious plan diffs whenever the API
response order differs from the order declared in configuration.

## Considered Options

- **`map(string)`** — rejected: cannot represent duplicate-name tags; lossy.
- **`map(list(string))`** — can represent duplicate names, but inverts the natural API shape
  (`{tag, value}` pairs become nested lists); more complex to read and write in HCL.
- **`list({name, value})`** — preserves multi-value semantics, but ordering is meaningless for
  tags, causing spurious diffs when Zabbix returns elements in a different order.
- **`set({name, value})` (chosen)** — order-independent, deduplicated on the full `{name,
  value}` pair, and matches the flat `{tag, value}` shape the API uses.

## Consequences

- The same `{name, value}` pair cannot appear twice in a `tags` set (set semantics). This is
  correct: a duplicate tag on a single entity is never meaningful.
- Tag `value` is optional (defaults to `""`), so bare tags like `{ name = "env" }` are valid
  HCL without requiring an explicit `value = ""`.
- All future resources that expose Zabbix tags must use the same `SetNestedAttribute` shape to
  keep the provider consistent. A standalone `TagModel` struct should be shared across
  resources rather than redefined per-resource.
