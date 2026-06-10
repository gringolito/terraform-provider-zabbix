# Typed enums for client-package API integer discriminator fields

The Zabbix API uses integer constants as discriminators on several object types (e.g.
`idp_type` on user directories, `type` on media types). In the client package, these
constants are declared using a named Go type rather than bare `int64` or `int` constants.

## Rationale

A bare integer constant (`const IDPTypeLDAP int64 = 1`) is assignable to any `int64`
variable. The compiler will not catch a caller that passes the wrong family of constants
(e.g. mistakenly using a media type constant where an IDP type constant is required). A
named type (`type IDPType int64`) makes such mismatches a compile error and makes the API
of client functions self-documenting — the signature `UserDirectoryGetByName(..., idpType
IDPType)` communicates intent; `(..., idpType int64)` does not.

## Considered Options

- **Bare typed constants (`const X int64 = …`)** — rejected: no compile-time protection
  against cross-family constant confusion; signatures are opaque.
- **Named type with typed constants (chosen)** — compiler enforces correct constant family
  at every callsite; function signatures are self-describing; zero runtime cost.

## Consequences

- Every API integer discriminator in the client package must use a named type. The struct
  field, function signatures, and constants all use the same type.
- The named type lives in the same file as the struct it describes (e.g. `IDPType` in
  `userdirectory.go`, `MediaTypeType` in `mediatype.go`).
- JSON marshalling is unaffected — `encoding/json` serialises named integer types
  identically to their underlying type.
