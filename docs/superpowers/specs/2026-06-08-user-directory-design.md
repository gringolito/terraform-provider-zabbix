# Design: `zabbix_user_directory_ldap` + `zabbix_user_directory_saml`

**Date:** 2026-06-08
**Issue:** [#21](https://github.com/gringolito/terraform-provider-zabbix/issues/21)
**Status:** Approved

## Overview

Implement two Terraform resources and two data sources for Zabbix user directories:
`zabbix_user_directory_ldap` and `zabbix_user_directory_saml`. Each maps to a single
Zabbix `userdirectory` API object (`idp_type` 1 and 2 respectively). Zabbix supports
multiple simultaneous directories of either type.

## Approach

**Option C — separate resources per subtype**, following the established
`zabbix_media_type_email` / `_script` / `_sms` / `_webhook` pattern. No discriminator
cross-field validation is needed; each resource exposes only its own fields.

## File Layout

```
internal/
  client/
    userdirectory.go           # UserDirectory struct + CRUD functions
    userdirectory_test.go      # unit tests via httptest server
  provider/
    user_directory_common.go   # shared schema builders, models, converters
    user_directory_ldap_resource.go
    user_directory_ldap_resource_test.go
    user_directory_ldap_data_source.go
    user_directory_ldap_data_source_test.go
    user_directory_saml_resource.go
    user_directory_saml_resource_test.go
    user_directory_saml_data_source.go
    user_directory_saml_data_source_test.go
examples/
  resources/
    zabbix_user_directory_ldap/resource.tf
    zabbix_user_directory_saml/resource.tf
  data-sources/
    zabbix_user_directory_ldap/data-source.tf
    zabbix_user_directory_saml/data-source.tf
```

Both resources and data sources registered in `internal/provider/provider.go`.

## Client Layer (`internal/client/userdirectory.go`)

Single `UserDirectory` struct covering all API fields. `BindPassword` is write-only
(Zabbix never returns it); the resource stores the configured value in state and never
overwrites it from the API read.

```go
type ProvisionGroup struct {
    Name       string `json:"name"`
    RoleID     string `json:"roleid"`
    UserGroups []struct {
        ID string `json:"usrgrpid"`
    } `json:"user_groups"`
}

type ProvisionMedia struct {
    MediaID     string `json:"userdirectory_mediaid,omitempty"`
    Name        string `json:"name"`
    MediaTypeID string `json:"mediatypeid"`
    Attribute   string `json:"attribute"`
    Active      int64  `json:"active,string"`
    Severity    int64  `json:"severity,string"`
    Period      string `json:"period"`
}

type UserDirectory struct {
    ID              string           `json:"userdirectoryid,omitempty"`
    IDPType         int64            `json:"idp_type,string"`
    Name            string           `json:"name"`
    Description     string           `json:"description"`
    ProvisionStatus int64            `json:"provision_status,string"`
    GroupName       string           `json:"group_name"`
    UserUsername    string           `json:"user_username"`
    UserLastname    string           `json:"user_lastname"`
    // LDAP-only
    Host            string `json:"host,omitempty"`
    Port            int64  `json:"port,string,omitempty"`
    BaseDN          string `json:"base_dn,omitempty"`
    SearchAttribute string `json:"search_attribute,omitempty"`
    BindDN          string `json:"bind_dn,omitempty"`
    BindPassword    string `json:"bind_password,omitempty"`
    StartTLS        int64  `json:"start_tls,string,omitempty"`
    SearchFilter    string `json:"search_filter,omitempty"`
    GroupBaseDN     string `json:"group_basedn,omitempty"`
    GroupMember     string `json:"group_member,omitempty"`
    GroupFilter     string `json:"group_filter,omitempty"`
    GroupMembership string `json:"group_membership,omitempty"`
    UserRefAttr     string `json:"user_ref_attr,omitempty"`
    // SAML-only
    IDPEntityID         string `json:"idp_entityid,omitempty"`
    SPEntityID          string `json:"sp_entityid,omitempty"`
    UsernameAttribute   string `json:"username_attribute,omitempty"`
    SSOURL              string `json:"sso_url,omitempty"`
    SLOURL              string `json:"slo_url,omitempty"`
    NameIDFormat        string `json:"nameid_format,omitempty"`
    SignMessages        int64  `json:"sign_messages,string,omitempty"`
    SignAssertions      int64  `json:"sign_assertions,string,omitempty"`
    SignAuthnRequests   int64  `json:"sign_authn_requests,string,omitempty"`
    SignLogoutRequests  int64  `json:"sign_logout_requests,string,omitempty"`
    SignLogoutResponses int64  `json:"sign_logout_responses,string,omitempty"`
    EncryptNameID       int64  `json:"encrypt_nameid,string,omitempty"`
    EncryptAssertions   int64  `json:"encrypt_assertions,string,omitempty"`
    SCIMStatus          int64  `json:"scim_status,string,omitempty"`
    // Always present
    ProvisionGroups []ProvisionGroup `json:"provision_groups"`
    ProvisionMedia  []ProvisionMedia `json:"provision_media"`
}
```

**Five client functions:**
- `UserDirectoryCreate(ctx, c, ud) (string, error)`
- `UserDirectoryGet(ctx, c, id) (*UserDirectory, error)` — includes `selectProvisionGroups: "extend"` and `selectProvisionMedia: "extend"`
- `UserDirectoryGetByName(ctx, c, name) ([]UserDirectory, error)`
- `UserDirectoryUpdate(ctx, c, ud) error`
- `UserDirectoryDelete(ctx, c, id) error`

## Common Layer (`internal/provider/user_directory_common.go`)

Shared models, schema builder functions, and converters — same role as `media_type_common.go`.

### Shared Models

```go
type UserDirectoryBaseModel struct {
    ID              types.String `tfsdk:"id"`
    Name            types.String `tfsdk:"name"`
    Description     types.String `tfsdk:"description"`
    ProvisionStatus types.String `tfsdk:"provision_status"`
    GroupName       types.String `tfsdk:"group_name"`
    UserUsername    types.String `tfsdk:"user_username"`
    UserLastname    types.String `tfsdk:"user_lastname"`
    ProvisionGroups types.List   `tfsdk:"provision_groups"`
    ProvisionMedia  types.List   `tfsdk:"provision_media"`
}

type ProvisionGroupModel struct {
    Name       types.String `tfsdk:"name"`
    RoleID     types.String `tfsdk:"role_id"`
    UserGroups types.Set    `tfsdk:"user_group_ids"` // Set of string IDs (unordered)
}

type ProvisionMediaModel struct {
    Name        types.String `tfsdk:"name"`
    MediaTypeID types.String `tfsdk:"media_type_id"`
    Attribute   types.String `tfsdk:"attribute"`
    Active      types.String `tfsdk:"active"`   // "enabled"/"disabled"
    Severity    types.Int64  `tfsdk:"severity"`
    Period      types.String `tfsdk:"period"`
}
```

### Schema Builders

- `commonUserDirectoryResourceAttributes() map[string]rschema.Attribute`
- `commonUserDirectoryDataSourceAttributes() map[string]dschema.Attribute`

### Converters

- `userDirectoryBaseFromModel(ctx, base) (client.UserDirectory, diag.Diagnostics)`
- `userDirectoryBaseToModel(ctx, ud, base) diag.Diagnostics`
- `lookupUserDirectory(ctx, c, idpType int64, id, name types.String) (*client.UserDirectory, diag.Diagnostics)` — handles id/name lookup, zero/multi-match errors, filters by `idp_type` to prevent cross-type name collision

## Resource Schemas

### `zabbix_user_directory_ldap`

```go
type UserDirectoryLDAPResourceModel struct {
    UserDirectoryBaseModel
    Host            types.String `tfsdk:"host"`
    Port            types.Int64  `tfsdk:"port"`
    BaseDN          types.String `tfsdk:"base_dn"`
    SearchAttribute types.String `tfsdk:"search_attribute"`
    BindDN          types.String `tfsdk:"bind_dn"`
    BindPassword    types.String `tfsdk:"bind_password"` // Sensitive, Optional, not Computed
    StartTLS        types.String `tfsdk:"start_tls"`     // "enabled"/"disabled"
    SearchFilter    types.String `tfsdk:"search_filter"`
    GroupBaseDN     types.String `tfsdk:"group_base_dn"`
    GroupMember     types.String `tfsdk:"group_member"`
    GroupFilter     types.String `tfsdk:"group_filter"`
    GroupMembership types.String `tfsdk:"group_membership"`
    UserRefAttr     types.String `tfsdk:"user_ref_attr"`
}
```

- Required: `name`, `host`, `base_dn`, `search_attribute`
- Defaults: `port` → `389`, `start_tls` → `"disabled"`
- `bind_password`: `Optional`, `Sensitive: true`, no `Computed` (never read back from API)

### `zabbix_user_directory_saml`

```go
type UserDirectorySAMLResourceModel struct {
    UserDirectoryBaseModel
    IDPEntityID         types.String `tfsdk:"idp_entityid"`
    SPEntityID          types.String `tfsdk:"sp_entityid"`
    UsernameAttribute   types.String `tfsdk:"username_attribute"`
    SSOURL              types.String `tfsdk:"sso_url"`
    SLOURL              types.String `tfsdk:"slo_url"`
    NameIDFormat        types.String `tfsdk:"nameid_format"`
    SignMessages        types.String `tfsdk:"sign_messages"`
    SignAssertions      types.String `tfsdk:"sign_assertions"`
    SignAuthnRequests   types.String `tfsdk:"sign_authn_requests"`
    SignLogoutRequests  types.String `tfsdk:"sign_logout_requests"`
    SignLogoutResponses types.String `tfsdk:"sign_logout_responses"`
    EncryptNameID       types.String `tfsdk:"encrypt_nameid"`
    EncryptAssertions   types.String `tfsdk:"encrypt_assertions"`
    SCIMStatus          types.String `tfsdk:"scim_status"`
}
```

- Required: `name`, `idp_entityid`, `sp_entityid`
- All boolean flags: `Optional+Computed`, default `"disabled"`, `stringvalidator.OneOf("enabled", "disabled")`

### Data Sources

Both data sources expose all resource attributes as `Computed`, with `id` and `name` as
`Optional+Computed` lookup keys. Pattern mirrors `role_data_source.go`.

## Testing

### Client Unit Tests (`userdirectory_test.go`)

Uses `httptest` server pattern from `role_test.go`:

- `TestUserDirectoryCreate_LDAP_Success`
- `TestUserDirectoryCreate_SAML_Success`
- `TestUserDirectoryGet_ReturnsNilWhenNotFound`
- `TestUserDirectoryGetByName_ReturnsMultiple`
- `TestUserDirectoryUpdate_Success`
- `TestUserDirectoryDelete_Success`
- `TestUserDirectoryCreate_ErrorEnvelope`

### Acceptance Tests

Uses `testhelper.Setup(t)` + `TF_ACC`, pattern from `role_resource_test.go`:

- `TestAccUserDirectoryLDAPResource_CRUD`
- `TestAccUserDirectoryLDAPResource_Import`
- `TestAccUserDirectoryLDAPResource_Drift_ProvisionGroups`
- `TestAccUserDirectorySAMLResource_CRUD`
- `TestAccUserDirectorySAMLResource_Import`
- `TestAccUserDirectoryLDAPDataSource_ByID`
- `TestAccUserDirectoryLDAPDataSource_ByName`
- `TestAccUserDirectorySAMLDataSource_ByID`
- `TestAccUserDirectorySAMLDataSource_ByName`

### Schema/Validator Unit Tests

No acceptance env needed; mirror `user_group_resource_schema_test.go`:

- LDAP: `host`, `base_dn`, `search_attribute` are required; `bind_password` is Sensitive
- SAML: `idp_entityid`, `sp_entityid` are required; all flag fields reject values outside `"enabled"`/`"disabled"`

## Out of Scope

- OIDC directory (not in Zabbix 7.0)
- SCIM-side IdP connector

## Pre-PR Checklist

`make generate` → `make build` → `make lint` → `make acc-tests`
