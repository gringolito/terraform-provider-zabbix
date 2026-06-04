# Translate API integer enums to human-readable strings in resource schemas

Whenever the Zabbix JSON-RPC API uses an integer to represent a finite set of known values
(e.g. `gui_access`, `debug_mode`, `users_status`, `smtp_security`), the provider schema exposes
the attribute as a **string** with human-readable values rather than exposing the raw integer.

## Rationale

Magic numbers are a bad user experience. A user reading `gui_access = 1` must look up the
Zabbix API docs to know that `1` means "internal authentication". A user reading
`gui_access = "internal"` does not. The schema becomes self-documenting without losing the
ability to drive any of the underlying API values.

The Zabbix API returns integers as JSON strings in 7.0 (e.g. `"gui_access": "0"`), so the
client layer already has to decode them. The extra translation step from integer to
human-readable string is negligible.

## Considered Options

- **Expose raw integers** — rejected: forces users to memorise Zabbix API constants; cannot be
  discovered from `terraform plan` output or IDE completion alone.
- **Translate to human-readable strings (chosen)** — schema is self-describing; valid values
  are enumerable in the `MarkdownDescription`; no runtime overhead beyond a map lookup.

## Consequences

- Each translated field requires a forward map (`string → int`) for writes and a reverse map
  (`int → string`) for reads in the provider layer; the client struct retains the integer type
  with `json:",string"` for the API decode.
- String values follow the Zabbix UI/documentation vocabulary in snake_case (e.g.
  `"system_default"`, `"normal_password"`, `"ssl_tls"`). When the API name is a single word,
  it is used as-is (e.g. `"internal"`, `"disabled"`, `"enabled"`).
- Valid values are rejected at plan time via `stringvalidator.OneOf`; `terraform validate`
  catches typos before any API call is made.
