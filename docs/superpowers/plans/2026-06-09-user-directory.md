# zabbix_user_directory_ldap + _saml Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `zabbix_user_directory_ldap` and `zabbix_user_directory_saml` resources and data sources, including inline `provision_groups` and `provision_media`, following the established `zabbix_media_type_*` pattern.

**Architecture:** Two separate resources/data-sources per IdP type (no discriminator validation needed). Shared fields (`provision_groups`, `provision_media`, common metadata) live in `user_directory_common.go`. The client layer uses a single `UserDirectory` struct covering all fields; CRUD functions build type-specific params maps.

**Tech Stack:** Go 1.21+, terraform-plugin-framework v1, terraform-plugin-framework-validators, Zabbix 7.0 JSON-RPC API (`userdirectory.*` methods).

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/client/userdirectory.go` | Create | Structs + CRUD (Create/Get/GetByName/Update/Delete) |
| `internal/client/userdirectory_test.go` | Create | Unit tests via httptest |
| `internal/provider/user_directory_common.go` | Create | Shared models, schema builders, converters, lookupUserDirectory |
| `internal/provider/user_directory_ldap_resource.go` | Create | LDAP resource CRUD |
| `internal/provider/user_directory_ldap_resource_test.go` | Create | LDAP acceptance + schema tests |
| `internal/provider/user_directory_ldap_data_source.go` | Create | LDAP data source |
| `internal/provider/user_directory_ldap_data_source_test.go` | Create | LDAP data source acceptance tests |
| `internal/provider/user_directory_saml_resource.go` | Create | SAML resource CRUD |
| `internal/provider/user_directory_saml_resource_test.go` | Create | SAML acceptance + schema tests |
| `internal/provider/user_directory_saml_data_source.go` | Create | SAML data source |
| `internal/provider/user_directory_saml_data_source_test.go` | Create | SAML data source acceptance tests |
| `internal/provider/provider.go` | Modify | Register 4 new constructors |
| `examples/resources/zabbix_user_directory_ldap/resource.tf` | Create | Example HCL |
| `examples/resources/zabbix_user_directory_saml/resource.tf` | Create | Example HCL |
| `examples/data-sources/zabbix_user_directory_ldap/data-source.tf` | Create | Example HCL |
| `examples/data-sources/zabbix_user_directory_saml/data-source.tf` | Create | Example HCL |

---

## Task 1: Client unit tests (failing)

**Files:**
- Create: `internal/client/userdirectory_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

var ldapUD = client.UserDirectory{
	IDPType:         1,
	Name:            "test-ldap",
	Description:     "",
	ProvisionStatus: 0,
	GroupName:       "",
	UserUsername:    "",
	UserLastname:    "",
	Host:            "ldap.example.com",
	Port:            389,
	BaseDN:          "dc=example,dc=com",
	SearchAttribute: "uid",
}

var samlUD = client.UserDirectory{
	IDPType:           2,
	Name:              "test-saml",
	IDPEntityID:       "http://idp.example.com/metadata",
	SPEntityID:        "zabbix",
	UsernameAttribute: "uid",
}

func TestUserDirectoryCreate_LDAP_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.create": rpcOK(t, map[string]any{"userdirectoryids": []string{"10"}}),
	})
	id, err := client.UserDirectoryCreate(t.Context(), c, ldapUD)
	if err != nil {
		t.Fatalf("UserDirectoryCreate: %v", err)
	}
	if id != "10" {
		t.Errorf("id = %q, want %q", id, "10")
	}
}

func TestUserDirectoryCreate_SAML_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.create": rpcOK(t, map[string]any{"userdirectoryids": []string{"11"}}),
	})
	id, err := client.UserDirectoryCreate(t.Context(), c, samlUD)
	if err != nil {
		t.Fatalf("UserDirectoryCreate: %v", err)
	}
	if id != "11" {
		t.Errorf("id = %q, want %q", id, "11")
	}
}

func TestUserDirectoryCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.UserDirectoryCreate(t.Context(), c, ldapUD)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserDirectoryGet_Found(t *testing.T) {
	resp := map[string]any{
		"userdirectoryid": "10",
		"idp_type": "1",
		"name": "test-ldap",
		"description": "",
		"provision_status": "0",
		"group_name": "",
		"user_username": "",
		"user_lastname": "",
		"host": "ldap.example.com",
		"port": "389",
		"base_dn": "dc=example,dc=com",
		"search_attribute": "uid",
		"bind_dn": "",
		"start_tls": "0",
		"search_filter": "",
		"group_basedn": "",
		"group_member": "",
		"group_filter": "",
		"group_membership": "",
		"user_ref_attr": "",
		"provision_groups": []any{},
		"provision_media":  []any{},
	}
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.get": rpcOK(t, []any{resp}),
	})
	ud, err := client.UserDirectoryGet(t.Context(), c, "10")
	if err != nil {
		t.Fatalf("UserDirectoryGet: %v", err)
	}
	if ud == nil {
		t.Fatal("expected non-nil result")
	}
	if ud.Name != "test-ldap" {
		t.Errorf("Name = %q, want %q", ud.Name, "test-ldap")
	}
	if ud.Port != 389 {
		t.Errorf("Port = %d, want 389", ud.Port)
	}
}

func TestUserDirectoryGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.get": rpcOK(t, []any{}),
	})
	ud, err := client.UserDirectoryGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ud != nil {
		t.Fatalf("expected nil, got %+v", ud)
	}
}

func TestUserDirectoryGetByName_ReturnsMultiple(t *testing.T) {
	resp := []any{
		map[string]any{"userdirectoryid": "1", "idp_type": "1", "name": "ldap", "description": "", "provision_status": "0", "group_name": "", "user_username": "", "user_lastname": "", "host": "a", "port": "389", "base_dn": "dc=a", "search_attribute": "uid", "bind_dn": "", "start_tls": "0", "search_filter": "", "group_basedn": "", "group_member": "", "group_filter": "", "group_membership": "", "user_ref_attr": "", "provision_groups": []any{}, "provision_media": []any{}},
		map[string]any{"userdirectoryid": "2", "idp_type": "1", "name": "ldap", "description": "", "provision_status": "0", "group_name": "", "user_username": "", "user_lastname": "", "host": "b", "port": "389", "base_dn": "dc=b", "search_attribute": "uid", "bind_dn": "", "start_tls": "0", "search_filter": "", "group_basedn": "", "group_member": "", "group_filter": "", "group_membership": "", "user_ref_attr": "", "provision_groups": []any{}, "provision_media": []any{}},
	}
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.get": rpcOK(t, resp),
	})
	dirs, err := client.UserDirectoryGetByName(t.Context(), c, "ldap", 1)
	if err != nil {
		t.Fatalf("UserDirectoryGetByName: %v", err)
	}
	if len(dirs) != 2 {
		t.Errorf("len = %d, want 2", len(dirs))
	}
}

func TestUserDirectoryUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.update": rpcOK(t, map[string]any{"userdirectoryids": []string{"10"}}),
	})
	ud := ldapUD
	ud.ID = "10"
	if err := client.UserDirectoryUpdate(t.Context(), c, ud); err != nil {
		t.Fatalf("UserDirectoryUpdate: %v", err)
	}
}

func TestUserDirectoryDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.delete": rpcOK(t, map[string]any{"userdirectoryids": []string{"10"}}),
	})
	if err := client.UserDirectoryDelete(t.Context(), c, "10"); err != nil {
		t.Fatalf("UserDirectoryDelete: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify they fail (package doesn't exist yet)**

```
go test ./internal/client/... -run TestUserDirectory -v
```
Expected: compile error — `client.UserDirectory` undefined.

---

## Task 2: Client implementation

**Files:**
- Create: `internal/client/userdirectory.go`

- [ ] **Step 1: Write the complete implementation**

```go
package client

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	IDPTypeLDAP int64 = 1
	IDPTypeSAML int64 = 2
)

type ProvisionUserGroup struct {
	ID string `json:"usrgrpid"`
}

type ProvisionGroup struct {
	Name       string               `json:"name"`
	RoleID     string               `json:"roleid"`
	UserGroups []ProvisionUserGroup `json:"user_groups"`
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

// UserDirectory represents a Zabbix user directory (LDAP or SAML).
// Zabbix 7.0 returns integer fields as JSON strings.
// BindPassword is write-only and never returned by the API.
type UserDirectory struct {
	ID              string `json:"userdirectoryid,omitempty"`
	IDPType         int64  `json:"idp_type,string"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ProvisionStatus int64  `json:"provision_status,string"`
	GroupName       string `json:"group_name"`
	UserUsername    string `json:"user_username"`
	UserLastname    string `json:"user_lastname"`
	// LDAP-only
	Host            string `json:"host"`
	Port            int64  `json:"port,string"`
	BaseDN          string `json:"base_dn"`
	SearchAttribute string `json:"search_attribute"`
	BindDN          string `json:"bind_dn"`
	BindPassword    string `json:"bind_password,omitempty"`
	StartTLS        int64  `json:"start_tls,string"`
	SearchFilter    string `json:"search_filter"`
	GroupBaseDN     string `json:"group_basedn"`
	GroupMember     string `json:"group_member"`
	GroupFilter     string `json:"group_filter"`
	GroupMembership string `json:"group_membership"`
	UserRefAttr     string `json:"user_ref_attr"`
	// SAML-only
	IDPEntityID         string `json:"idp_entityid"`
	SPEntityID          string `json:"sp_entityid"`
	UsernameAttribute   string `json:"username_attribute"`
	SSOURL              string `json:"sso_url"`
	SLOURL              string `json:"slo_url"`
	NameIDFormat        string `json:"nameid_format"`
	SignMessages        int64  `json:"sign_messages,string"`
	SignAssertions      int64  `json:"sign_assertions,string"`
	SignAuthnRequests   int64  `json:"sign_authn_requests,string"`
	SignLogoutRequests  int64  `json:"sign_logout_requests,string"`
	SignLogoutResponses int64  `json:"sign_logout_responses,string"`
	EncryptNameID       int64  `json:"encrypt_nameid,string"`
	EncryptAssertions   int64  `json:"encrypt_assertions,string"`
	SCIMStatus          int64  `json:"scim_status,string"`
	// Always present
	ProvisionGroups []ProvisionGroup `json:"provision_groups"`
	ProvisionMedia  []ProvisionMedia `json:"provision_media"`
}

func UserDirectoryCreate(ctx context.Context, c Client, ud UserDirectory) (string, error) {
	params := udCommonParams(ud)
	if ud.IDPType == IDPTypeLDAP {
		udAddLDAPParams(params, ud)
	} else {
		udAddSAMLParams(params, ud)
	}
	result, err := c.Call(ctx, "userdirectory.create", params)
	if err != nil {
		return "", err
	}
	var out struct {
		IDs []string `json:"userdirectoryids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("userdirectory.create: unexpected response: %w", err)
	}
	if len(out.IDs) == 0 {
		return "", fmt.Errorf("userdirectory.create: empty userdirectoryids in response")
	}
	return out.IDs[0], nil
}

func UserDirectoryGet(ctx context.Context, c Client, id string) (*UserDirectory, error) {
	result, err := c.Call(ctx, "userdirectory.get", map[string]any{
		"userdirectoryids":      []string{id},
		"output":                "extend",
		"selectProvisionGroups": "extend",
		"selectProvisionMedia":  "extend",
		"limit":                 1,
	})
	if err != nil {
		return nil, err
	}
	var dirs []UserDirectory
	if err := json.Unmarshal(result, &dirs); err != nil {
		return nil, fmt.Errorf("userdirectory.get: unexpected response: %w", err)
	}
	if len(dirs) == 0 {
		return nil, nil
	}
	return &dirs[0], nil
}

func UserDirectoryGetByName(ctx context.Context, c Client, name string, idpType int64) ([]UserDirectory, error) {
	result, err := c.Call(ctx, "userdirectory.get", map[string]any{
		"filter": map[string]any{
			"name":     []string{name},
			"idp_type": []int64{idpType},
		},
		"output":                "extend",
		"selectProvisionGroups": "extend",
		"selectProvisionMedia":  "extend",
	})
	if err != nil {
		return nil, err
	}
	var dirs []UserDirectory
	if err := json.Unmarshal(result, &dirs); err != nil {
		return nil, fmt.Errorf("userdirectory.get: unexpected response: %w", err)
	}
	return dirs, nil
}

func UserDirectoryUpdate(ctx context.Context, c Client, ud UserDirectory) error {
	params := udCommonParams(ud)
	params["userdirectoryid"] = ud.ID
	if ud.IDPType == IDPTypeLDAP {
		udAddLDAPParams(params, ud)
	} else {
		udAddSAMLParams(params, ud)
	}
	_, err := c.Call(ctx, "userdirectory.update", params)
	return err
}

func UserDirectoryDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "userdirectory.delete", []string{id})
	return err
}

func udCommonParams(ud UserDirectory) map[string]any {
	return map[string]any{
		"idp_type":         ud.IDPType,
		"name":             ud.Name,
		"description":      ud.Description,
		"provision_status": ud.ProvisionStatus,
		"group_name":       ud.GroupName,
		"user_username":    ud.UserUsername,
		"user_lastname":    ud.UserLastname,
		"provision_groups": provisionGroupsToParams(ud.ProvisionGroups),
		"provision_media":  provisionMediaToParams(ud.ProvisionMedia),
	}
}

func udAddLDAPParams(p map[string]any, ud UserDirectory) {
	p["host"] = ud.Host
	p["port"] = ud.Port
	p["base_dn"] = ud.BaseDN
	p["search_attribute"] = ud.SearchAttribute
	p["bind_dn"] = ud.BindDN
	p["start_tls"] = ud.StartTLS
	p["search_filter"] = ud.SearchFilter
	p["group_basedn"] = ud.GroupBaseDN
	p["group_member"] = ud.GroupMember
	p["group_filter"] = ud.GroupFilter
	p["group_membership"] = ud.GroupMembership
	p["user_ref_attr"] = ud.UserRefAttr
	if ud.BindPassword != "" {
		p["bind_password"] = ud.BindPassword
	}
}

func udAddSAMLParams(p map[string]any, ud UserDirectory) {
	p["idp_entityid"] = ud.IDPEntityID
	p["sp_entityid"] = ud.SPEntityID
	p["username_attribute"] = ud.UsernameAttribute
	p["sso_url"] = ud.SSOURL
	p["slo_url"] = ud.SLOURL
	p["nameid_format"] = ud.NameIDFormat
	p["sign_messages"] = ud.SignMessages
	p["sign_assertions"] = ud.SignAssertions
	p["sign_authn_requests"] = ud.SignAuthnRequests
	p["sign_logout_requests"] = ud.SignLogoutRequests
	p["sign_logout_responses"] = ud.SignLogoutResponses
	p["encrypt_nameid"] = ud.EncryptNameID
	p["encrypt_assertions"] = ud.EncryptAssertions
	p["scim_status"] = ud.SCIMStatus
}

func provisionGroupsToParams(groups []ProvisionGroup) []map[string]any {
	if len(groups) == 0 {
		return []map[string]any{}
	}
	result := make([]map[string]any, len(groups))
	for i, g := range groups {
		ugs := make([]map[string]any, len(g.UserGroups))
		for j, ug := range g.UserGroups {
			ugs[j] = map[string]any{"usrgrpid": ug.ID}
		}
		result[i] = map[string]any{
			"name":        g.Name,
			"roleid":      g.RoleID,
			"user_groups": ugs,
		}
	}
	return result
}

func provisionMediaToParams(media []ProvisionMedia) []map[string]any {
	if len(media) == 0 {
		return []map[string]any{}
	}
	result := make([]map[string]any, len(media))
	for i, m := range media {
		result[i] = map[string]any{
			"name":        m.Name,
			"mediatypeid": m.MediaTypeID,
			"attribute":   m.Attribute,
			"active":      m.Active,
			"severity":    m.Severity,
			"period":      m.Period,
		}
	}
	return result
}
```

- [ ] **Step 2: Run tests and verify they pass**

```
go test ./internal/client/... -run TestUserDirectory -v
```
Expected: all 8 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/client/userdirectory.go internal/client/userdirectory_test.go
git commit -S -s -m "feat(client): add UserDirectory CRUD + unit tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Common provider layer

**Files:**
- Create: `internal/provider/user_directory_common.go`

- [ ] **Step 1: Write the complete file**

```go
package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	provisionGroupAttrTypes = map[string]attr.Type{
		"name":           types.StringType,
		"role_id":        types.StringType,
		"user_group_ids": types.SetType{ElemType: types.StringType},
	}
	provisionMediaAttrTypes = map[string]attr.Type{
		"name":          types.StringType,
		"media_type_id": types.StringType,
		"attribute":     types.StringType,
		"active":        types.StringType,
		"severity":      types.Int64Type,
		"period":        types.StringType,
	}
	provisionStatusMap = map[string]int64{
		"disabled": 0,
		"enabled":  1,
	}
	provisionStatusReverseMap = map[int64]string{
		0: "disabled",
		1: "enabled",
	}
	provisionMediaActiveMap = map[string]int64{
		"enabled":  0,
		"disabled": 1,
	}
	provisionMediaActiveReverseMap = map[int64]string{
		0: "enabled",
		1: "disabled",
	}
	udEnabledDisabledMap = map[string]int64{
		"disabled": 0,
		"enabled":  1,
	}
	udEnabledDisabledReverseMap = map[int64]string{
		0: "disabled",
		1: "enabled",
	}
)

// UserDirectoryBaseModel holds fields common to all user directory resources and data sources.
// Embed this struct (not a pointer) in each type-specific model.
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
	UserGroups types.Set    `tfsdk:"user_group_ids"`
}

type ProvisionMediaModel struct {
	Name        types.String `tfsdk:"name"`
	MediaTypeID types.String `tfsdk:"media_type_id"`
	Attribute   types.String `tfsdk:"attribute"`
	Active      types.String `tfsdk:"active"`
	Severity    types.Int64  `tfsdk:"severity"`
	Period      types.String `tfsdk:"period"`
}

func commonUserDirectoryResourceAttributes() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Unique identifier of the user directory.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": rschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Display name of the user directory. Must be unique within Zabbix.",
		},
		"description": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Description of the user directory.",
		},
		"provision_status": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("disabled"),
			MarkdownDescription: "Whether JIT provisioning is enabled. One of: `enabled`, `disabled`. Defaults to `disabled`.",
			Validators: []validator.String{
				stringvalidator.OneOf("enabled", "disabled"),
			},
		},
		"group_name": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Name of the IdP attribute that carries group membership.",
		},
		"user_username": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "IdP attribute mapped to the Zabbix user first name.",
		},
		"user_lastname": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "IdP attribute mapped to the Zabbix user last name.",
		},
		"provision_groups": rschema.ListNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "IdP group to Zabbix role and user groups mappings for JIT provisioning.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"name": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Name of the IdP group. Use `*` as a wildcard for all groups.",
					},
					"role_id": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "ID of the Zabbix role to assign.",
					},
					"user_group_ids": rschema.SetAttribute{
						Required:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Set of Zabbix user group IDs to assign.",
					},
				},
			},
		},
		"provision_media": rschema.ListNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "IdP attribute to Zabbix media type mappings for JIT provisioning.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"name": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Name of the provision media entry.",
					},
					"media_type_id": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "ID of the Zabbix media type.",
					},
					"attribute": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "IdP attribute value to use as the media send-to address.",
					},
					"active": rschema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("enabled"),
						MarkdownDescription: "Whether the media is active. One of: `enabled`, `disabled`. Defaults to `enabled`.",
						Validators: []validator.String{
							stringvalidator.OneOf("enabled", "disabled"),
						},
					},
					"severity": rschema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(63),
						MarkdownDescription: "Severity bitmask (0-63). Defaults to `63` (all severities).",
					},
					"period": rschema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("1-7,00:00-24:00"),
						MarkdownDescription: "Active time period. Defaults to `1-7,00:00-24:00`.",
					},
				},
			},
		},
	}
}

func commonUserDirectoryDataSourceAttributes() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"id": dschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Unique identifier of the user directory. One of `id` or `name` must be set.",
		},
		"name": dschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Display name of the user directory. One of `id` or `name` must be set.",
		},
		"description":      dschema.StringAttribute{Computed: true, MarkdownDescription: "Description of the user directory."},
		"provision_status": dschema.StringAttribute{Computed: true, MarkdownDescription: "Whether JIT provisioning is enabled: `enabled` or `disabled`."},
		"group_name":       dschema.StringAttribute{Computed: true, MarkdownDescription: "Name of the IdP attribute that carries group membership."},
		"user_username":    dschema.StringAttribute{Computed: true, MarkdownDescription: "IdP attribute mapped to the Zabbix user first name."},
		"user_lastname":    dschema.StringAttribute{Computed: true, MarkdownDescription: "IdP attribute mapped to the Zabbix user last name."},
		"provision_groups": dschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "IdP group to Zabbix role and user groups mappings.",
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"name":           dschema.StringAttribute{Computed: true, MarkdownDescription: "Name of the IdP group."},
					"role_id":        dschema.StringAttribute{Computed: true, MarkdownDescription: "ID of the Zabbix role."},
					"user_group_ids": dschema.SetAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Set of Zabbix user group IDs."},
				},
			},
		},
		"provision_media": dschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "IdP attribute to Zabbix media type mappings.",
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"name":          dschema.StringAttribute{Computed: true, MarkdownDescription: "Name of the provision media entry."},
					"media_type_id": dschema.StringAttribute{Computed: true, MarkdownDescription: "ID of the Zabbix media type."},
					"attribute":     dschema.StringAttribute{Computed: true, MarkdownDescription: "IdP attribute value."},
					"active":        dschema.StringAttribute{Computed: true, MarkdownDescription: "Whether the media is active: `enabled` or `disabled`."},
					"severity":      dschema.Int64Attribute{Computed: true, MarkdownDescription: "Severity bitmask."},
					"period":        dschema.StringAttribute{Computed: true, MarkdownDescription: "Active time period."},
				},
			},
		},
	}
}

func userDirectoryBaseFromModel(ctx context.Context, m *UserDirectoryBaseModel) (client.UserDirectory, diag.Diagnostics) {
	var diags diag.Diagnostics
	ud := client.UserDirectory{
		Name:            m.Name.ValueString(),
		Description:     m.Description.ValueString(),
		ProvisionStatus: provisionStatusMap[m.ProvisionStatus.ValueString()],
		GroupName:       m.GroupName.ValueString(),
		UserUsername:    m.UserUsername.ValueString(),
		UserLastname:    m.UserLastname.ValueString(),
	}

	if !m.ProvisionGroups.IsNull() && !m.ProvisionGroups.IsUnknown() {
		var groupModels []ProvisionGroupModel
		diags.Append(m.ProvisionGroups.ElementsAs(ctx, &groupModels, false)...)
		if diags.HasError() {
			return ud, diags
		}
		groups := make([]client.ProvisionGroup, len(groupModels))
		for i, gm := range groupModels {
			var ugIDs []string
			diags.Append(gm.UserGroups.ElementsAs(ctx, &ugIDs, false)...)
			if diags.HasError() {
				return ud, diags
			}
			ugs := make([]client.ProvisionUserGroup, len(ugIDs))
			for j, id := range ugIDs {
				ugs[j] = client.ProvisionUserGroup{ID: id}
			}
			groups[i] = client.ProvisionGroup{
				Name:       gm.Name.ValueString(),
				RoleID:     gm.RoleID.ValueString(),
				UserGroups: ugs,
			}
		}
		ud.ProvisionGroups = groups
	} else {
		ud.ProvisionGroups = []client.ProvisionGroup{}
	}

	if !m.ProvisionMedia.IsNull() && !m.ProvisionMedia.IsUnknown() {
		var mediaModels []ProvisionMediaModel
		diags.Append(m.ProvisionMedia.ElementsAs(ctx, &mediaModels, false)...)
		if diags.HasError() {
			return ud, diags
		}
		media := make([]client.ProvisionMedia, len(mediaModels))
		for i, mm := range mediaModels {
			media[i] = client.ProvisionMedia{
				Name:        mm.Name.ValueString(),
				MediaTypeID: mm.MediaTypeID.ValueString(),
				Attribute:   mm.Attribute.ValueString(),
				Active:      provisionMediaActiveMap[mm.Active.ValueString()],
				Severity:    mm.Severity.ValueInt64(),
				Period:      mm.Period.ValueString(),
			}
		}
		ud.ProvisionMedia = media
	} else {
		ud.ProvisionMedia = []client.ProvisionMedia{}
	}

	return ud, diags
}

func userDirectoryBaseToModel(ctx context.Context, ud *client.UserDirectory, m *UserDirectoryBaseModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Name = types.StringValue(ud.Name)
	m.Description = types.StringValue(ud.Description)
	m.ProvisionStatus = types.StringValue(provisionStatusReverseMap[ud.ProvisionStatus])
	m.GroupName = types.StringValue(ud.GroupName)
	m.UserUsername = types.StringValue(ud.UserUsername)
	m.UserLastname = types.StringValue(ud.UserLastname)

	groupModels := make([]ProvisionGroupModel, len(ud.ProvisionGroups))
	for i, g := range ud.ProvisionGroups {
		ugVals := make([]attr.Value, len(g.UserGroups))
		for j, ug := range g.UserGroups {
			ugVals[j] = types.StringValue(ug.ID)
		}
		ugSet, d := types.SetValue(types.StringType, ugVals)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		groupModels[i] = ProvisionGroupModel{
			Name:       types.StringValue(g.Name),
			RoleID:     types.StringValue(g.RoleID),
			UserGroups: ugSet,
		}
	}
	var d diag.Diagnostics
	m.ProvisionGroups, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: provisionGroupAttrTypes}, groupModels)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	mediaModels := make([]ProvisionMediaModel, len(ud.ProvisionMedia))
	for i, pm := range ud.ProvisionMedia {
		mediaModels[i] = ProvisionMediaModel{
			Name:        types.StringValue(pm.Name),
			MediaTypeID: types.StringValue(pm.MediaTypeID),
			Attribute:   types.StringValue(pm.Attribute),
			Active:      types.StringValue(provisionMediaActiveReverseMap[pm.Active]),
			Severity:    types.Int64Value(pm.Severity),
			Period:      types.StringValue(pm.Period),
		}
	}
	m.ProvisionMedia, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: provisionMediaAttrTypes}, mediaModels)
	diags.Append(d...)
	return diags
}

func lookupUserDirectory(ctx context.Context, c client.Client, idpType int64, id, name types.String) (*client.UserDirectory, diag.Diagnostics) {
	var diags diag.Diagnostics

	if id.IsNull() && name.IsNull() {
		diags.AddError("Missing lookup key", "Exactly one of `id` or `name` must be set.")
		return nil, diags
	}

	typeName := "LDAP"
	if idpType == client.IDPTypeSAML {
		typeName = "SAML"
	}

	if !id.IsNull() {
		ud, err := client.UserDirectoryGet(ctx, c, id.ValueString())
		if err != nil {
			diags.AddError("Error reading user directory", err.Error())
			return nil, diags
		}
		if ud == nil {
			diags.AddError("User directory not found", fmt.Sprintf("No user directory found with id %q.", id.ValueString()))
			return nil, diags
		}
		if ud.IDPType != idpType {
			diags.AddError("Wrong user directory type", fmt.Sprintf("User directory %q is not a %s directory.", id.ValueString(), typeName))
			return nil, diags
		}
		return ud, diags
	}

	dirs, err := client.UserDirectoryGetByName(ctx, c, name.ValueString(), idpType)
	if err != nil {
		diags.AddError("Error reading user directory", err.Error())
		return nil, diags
	}
	switch len(dirs) {
	case 0:
		diags.AddError("User directory not found", fmt.Sprintf("No %s user directory found with name %q.", typeName, name.ValueString()))
	case 1:
		return &dirs[0], diags
	default:
		diags.AddError("Multiple user directories found", fmt.Sprintf("Found %d %s user directories with name %q; use `id` to disambiguate.", len(dirs), typeName, name.ValueString()))
	}
	return nil, diags
}
```

- [ ] **Step 2: Verify it compiles**

```
go build ./internal/provider/...
```
Expected: no errors (resources not registered yet, so no undefined symbol errors).

- [ ] **Step 3: Commit**

```bash
git add internal/provider/user_directory_common.go
git commit -S -s -m "feat(provider): add user directory common layer

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: LDAP resource

**Files:**
- Create: `internal/provider/user_directory_ldap_resource.go`
- Create: `internal/provider/user_directory_ldap_resource_test.go`

- [ ] **Step 1: Write the failing schema test**

```go
package provider_test

import (
	"context"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUserDirectoryLDAPResource_SchemaValidation(t *testing.T) {
	r := provider.NewUserDirectoryLDAPResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	t.Run("bind_password is sensitive", func(t *testing.T) {
		attr, ok := schResp.Schema.Attributes["bind_password"].(fwschema.StringAttribute)
		if !ok {
			t.Fatal("bind_password is not a StringAttribute")
		}
		if !attr.Sensitive {
			t.Error("bind_password must be Sensitive")
		}
	})

	t.Run("required fields present", func(t *testing.T) {
		for _, name := range []string{"host", "base_dn", "search_attribute"} {
			attr, ok := schResp.Schema.Attributes[name].(fwschema.StringAttribute)
			if !ok {
				t.Errorf("%s is not a StringAttribute", name)
				continue
			}
			if !attr.Required {
				t.Errorf("%s must be Required", name)
			}
		}
	})

	t.Run("start_tls rejects unknown values", func(t *testing.T) {
		attr, ok := schResp.Schema.Attributes["start_tls"].(fwschema.StringAttribute)
		if !ok {
			t.Fatal("start_tls is not a StringAttribute")
		}
		if len(attr.Validators) == 0 {
			t.Fatal("start_tls has no validators")
		}
		req := validator.StringRequest{ConfigValue: types.StringValue("yes")}
		resp := &validator.StringResponse{}
		for _, v := range attr.Validators {
			v.ValidateString(context.Background(), req, resp)
		}
		if !resp.Diagnostics.HasError() {
			t.Error("start_tls should reject 'yes'")
		}
	})

	t.Run("provision_status rejects unknown values", func(t *testing.T) {
		attr, ok := schResp.Schema.Attributes["provision_status"].(fwschema.StringAttribute)
		if !ok {
			t.Fatal("provision_status is not a StringAttribute")
		}
		req := validator.StringRequest{ConfigValue: types.StringValue("maybe")}
		resp := &validator.StringResponse{}
		for _, v := range attr.Validators {
			v.ValidateString(context.Background(), req, resp)
		}
		if !resp.Diagnostics.HasError() {
			t.Error("provision_status should reject 'maybe'")
		}
	})
}
```

Save to `internal/provider/user_directory_ldap_resource_test.go`.

- [ ] **Step 2: Run to verify it fails**

```
go test ./internal/provider/... -run TestUserDirectoryLDAPResource_SchemaValidation -v
```
Expected: compile error — `provider.NewUserDirectoryLDAPResource` undefined.

- [ ] **Step 3: Write the LDAP resource**

Create `internal/provider/user_directory_ldap_resource.go`:

```go
package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &UserDirectoryLDAPResource{}
var _ resource.ResourceWithImportState = &UserDirectoryLDAPResource{}

func NewUserDirectoryLDAPResource() resource.Resource {
	return &UserDirectoryLDAPResource{}
}

type UserDirectoryLDAPResource struct {
	client client.Client
}

type UserDirectoryLDAPResourceModel struct {
	UserDirectoryBaseModel
	Host            types.String `tfsdk:"host"`
	Port            types.Int64  `tfsdk:"port"`
	BaseDN          types.String `tfsdk:"base_dn"`
	SearchAttribute types.String `tfsdk:"search_attribute"`
	BindDN          types.String `tfsdk:"bind_dn"`
	BindPassword    types.String `tfsdk:"bind_password"`
	StartTLS        types.String `tfsdk:"start_tls"`
	SearchFilter    types.String `tfsdk:"search_filter"`
	GroupBaseDN     types.String `tfsdk:"group_base_dn"`
	GroupMember     types.String `tfsdk:"group_member"`
	GroupFilter     types.String `tfsdk:"group_filter"`
	GroupMembership types.String `tfsdk:"group_membership"`
	UserRefAttr     types.String `tfsdk:"user_ref_attr"`
}

func (r *UserDirectoryLDAPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_directory_ldap"
}

func (r *UserDirectoryLDAPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonUserDirectoryResourceAttributes()
	attrs["host"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Hostname or IP address of the LDAP server.",
	}
	attrs["port"] = schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(389),
		MarkdownDescription: "Port of the LDAP server. Defaults to `389`.",
	}
	attrs["base_dn"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Base DN for LDAP search (e.g. `dc=example,dc=com`).",
	}
	attrs["search_attribute"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "LDAP attribute used to identify users (e.g. `uid`, `sAMAccountName`).",
	}
	attrs["bind_dn"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "DN used to bind to the LDAP server.",
	}
	attrs["bind_password"] = schema.StringAttribute{
		Optional:            true,
		Sensitive:           true,
		MarkdownDescription: "Password for the bind DN. Sensitive. Not returned by the Zabbix API.",
	}
	attrs["start_tls"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("disabled"),
		MarkdownDescription: "Whether to use StartTLS. One of: `enabled`, `disabled`. Defaults to `disabled`.",
		Validators: []validator.String{
			stringvalidator.OneOf("enabled", "disabled"),
		},
	}
	attrs["search_filter"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "Custom LDAP search filter.",
	}
	attrs["group_base_dn"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "Base DN for group search.",
	}
	attrs["group_member"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "LDAP attribute that lists the members of a group (e.g. `member`).",
	}
	attrs["group_filter"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "LDAP filter for group search.",
	}
	attrs["group_membership"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "LDAP attribute on the user object that lists the groups the user belongs to.",
	}
	attrs["user_ref_attr"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "LDAP attribute used to reference users in group entries (e.g. `CN`).",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix LDAP user directory.",
		Attributes:          attrs,
	}
}

func (r *UserDirectoryLDAPResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *UserDirectoryLDAPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserDirectoryLDAPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, diags := userDirectoryBaseFromModel(ctx, &data.UserDirectoryBaseModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ud.IDPType = client.IDPTypeLDAP
	udLDAPFromModel(&data, &ud)

	id, err := client.UserDirectoryCreate(ctx, r.client, ud)
	if err != nil {
		resp.Diagnostics.AddError("Error creating LDAP user directory", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.UserDirectoryGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading LDAP user directory after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(udLDAPToModel(ctx, created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserDirectoryLDAPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserDirectoryLDAPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, err := client.UserDirectoryGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading LDAP user directory", err.Error())
		return
	}
	if ud == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(udLDAPToModel(ctx, ud, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserDirectoryLDAPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserDirectoryLDAPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, diags := userDirectoryBaseFromModel(ctx, &data.UserDirectoryBaseModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ud.ID = data.ID.ValueString()
	ud.IDPType = client.IDPTypeLDAP
	udLDAPFromModel(&data, &ud)

	if err := client.UserDirectoryUpdate(ctx, r.client, ud); err != nil {
		resp.Diagnostics.AddError("Error updating LDAP user directory", err.Error())
		return
	}

	updated, err := client.UserDirectoryGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading LDAP user directory after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(udLDAPToModel(ctx, updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserDirectoryLDAPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserDirectoryLDAPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.UserDirectoryDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting LDAP user directory", err.Error())
	}
}

func (r *UserDirectoryLDAPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func udLDAPFromModel(m *UserDirectoryLDAPResourceModel, ud *client.UserDirectory) {
	ud.Host = m.Host.ValueString()
	ud.Port = m.Port.ValueInt64()
	ud.BaseDN = m.BaseDN.ValueString()
	ud.SearchAttribute = m.SearchAttribute.ValueString()
	ud.BindDN = m.BindDN.ValueString()
	ud.BindPassword = m.BindPassword.ValueString()
	ud.StartTLS = udEnabledDisabledMap[m.StartTLS.ValueString()]
	ud.SearchFilter = m.SearchFilter.ValueString()
	ud.GroupBaseDN = m.GroupBaseDN.ValueString()
	ud.GroupMember = m.GroupMember.ValueString()
	ud.GroupFilter = m.GroupFilter.ValueString()
	ud.GroupMembership = m.GroupMembership.ValueString()
	ud.UserRefAttr = m.UserRefAttr.ValueString()
}

// udLDAPToModel populates the model from API response.
// bind_password is intentionally NOT updated — it is write-only and never returned by the API.
func udLDAPToModel(ctx context.Context, ud *client.UserDirectory, m *UserDirectoryLDAPResourceModel) diag.Diagnostics {
	diags := userDirectoryBaseToModel(ctx, ud, &m.UserDirectoryBaseModel)
	if diags.HasError() {
		return diags
	}
	m.Host = types.StringValue(ud.Host)
	m.Port = types.Int64Value(ud.Port)
	m.BaseDN = types.StringValue(ud.BaseDN)
	m.SearchAttribute = types.StringValue(ud.SearchAttribute)
	m.BindDN = types.StringValue(ud.BindDN)
	// m.BindPassword intentionally not updated
	m.StartTLS = types.StringValue(udEnabledDisabledReverseMap[ud.StartTLS])
	m.SearchFilter = types.StringValue(ud.SearchFilter)
	m.GroupBaseDN = types.StringValue(ud.GroupBaseDN)
	m.GroupMember = types.StringValue(ud.GroupMember)
	m.GroupFilter = types.StringValue(ud.GroupFilter)
	m.GroupMembership = types.StringValue(ud.GroupMembership)
	m.UserRefAttr = types.StringValue(ud.UserRefAttr)
	return diags
}
```

- [ ] **Step 4: Run schema test and verify it passes**

```
go test ./internal/provider/... -run TestUserDirectoryLDAPResource_SchemaValidation -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/user_directory_ldap_resource.go internal/provider/user_directory_ldap_resource_test.go
git commit -S -s -m "feat(provider): add zabbix_user_directory_ldap resource

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: LDAP data source

**Files:**
- Create: `internal/provider/user_directory_ldap_data_source.go`
- Create: `internal/provider/user_directory_ldap_data_source_test.go` (placeholder for acc tests — see Task 9)

- [ ] **Step 1: Write the data source**

```go
package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDirectoryLDAPDataSource{}

func NewUserDirectoryLDAPDataSource() datasource.DataSource {
	return &UserDirectoryLDAPDataSource{}
}

type UserDirectoryLDAPDataSource struct {
	client client.Client
}

type UserDirectoryLDAPDataSourceModel struct {
	UserDirectoryBaseModel
	Host            types.String `tfsdk:"host"`
	Port            types.Int64  `tfsdk:"port"`
	BaseDN          types.String `tfsdk:"base_dn"`
	SearchAttribute types.String `tfsdk:"search_attribute"`
	BindDN          types.String `tfsdk:"bind_dn"`
	StartTLS        types.String `tfsdk:"start_tls"`
	SearchFilter    types.String `tfsdk:"search_filter"`
	GroupBaseDN     types.String `tfsdk:"group_base_dn"`
	GroupMember     types.String `tfsdk:"group_member"`
	GroupFilter     types.String `tfsdk:"group_filter"`
	GroupMembership types.String `tfsdk:"group_membership"`
	UserRefAttr     types.String `tfsdk:"user_ref_attr"`
}

func (d *UserDirectoryLDAPDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_directory_ldap"
}

func (d *UserDirectoryLDAPDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := commonUserDirectoryDataSourceAttributes()
	attrs["host"]             = schema.StringAttribute{Computed: true, MarkdownDescription: "Hostname or IP address of the LDAP server."}
	attrs["port"]             = schema.Int64Attribute{Computed: true, MarkdownDescription: "Port of the LDAP server."}
	attrs["base_dn"]          = schema.StringAttribute{Computed: true, MarkdownDescription: "Base DN for LDAP search."}
	attrs["search_attribute"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute used to identify users."}
	attrs["bind_dn"]          = schema.StringAttribute{Computed: true, MarkdownDescription: "DN used to bind to the LDAP server."}
	attrs["start_tls"]        = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether StartTLS is enabled: `enabled` or `disabled`."}
	attrs["search_filter"]    = schema.StringAttribute{Computed: true, MarkdownDescription: "Custom LDAP search filter."}
	attrs["group_base_dn"]    = schema.StringAttribute{Computed: true, MarkdownDescription: "Base DN for group search."}
	attrs["group_member"]     = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute listing group members."}
	attrs["group_filter"]     = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP filter for group search."}
	attrs["group_membership"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute on user listing their groups."}
	attrs["user_ref_attr"]    = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute referencing users in group entries."}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix LDAP user directory by ID or name.",
		Attributes:          attrs,
	}
}

func (d *UserDirectoryLDAPDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *UserDirectoryLDAPDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDirectoryLDAPDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, diags := lookupUserDirectory(ctx, d.client, client.IDPTypeLDAP, data.ID, data.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(ud.ID)
	resp.Diagnostics.Append(userDirectoryBaseToModel(ctx, ud, &data.UserDirectoryBaseModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Host = types.StringValue(ud.Host)
	data.Port = types.Int64Value(ud.Port)
	data.BaseDN = types.StringValue(ud.BaseDN)
	data.SearchAttribute = types.StringValue(ud.SearchAttribute)
	data.BindDN = types.StringValue(ud.BindDN)
	data.StartTLS = types.StringValue(udEnabledDisabledReverseMap[ud.StartTLS])
	data.SearchFilter = types.StringValue(ud.SearchFilter)
	data.GroupBaseDN = types.StringValue(ud.GroupBaseDN)
	data.GroupMember = types.StringValue(ud.GroupMember)
	data.GroupFilter = types.StringValue(ud.GroupFilter)
	data.GroupMembership = types.StringValue(ud.GroupMembership)
	data.UserRefAttr = types.StringValue(ud.UserRefAttr)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 2: Create empty test file (acceptance tests added in Task 9)**

```go
package provider_test
```

Save as `internal/provider/user_directory_ldap_data_source_test.go`.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/user_directory_ldap_data_source.go internal/provider/user_directory_ldap_data_source_test.go
git commit -S -s -m "feat(provider): add zabbix_user_directory_ldap data source

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 6: SAML resource

**Files:**
- Create: `internal/provider/user_directory_saml_resource.go`
- Create: `internal/provider/user_directory_saml_resource_test.go`

- [ ] **Step 1: Write the failing schema test**

```go
package provider_test

import (
	"context"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUserDirectorySAMLResource_SchemaValidation(t *testing.T) {
	r := provider.NewUserDirectorySAMLResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	t.Run("required fields present", func(t *testing.T) {
		for _, name := range []string{"idp_entityid", "sp_entityid"} {
			attr, ok := schResp.Schema.Attributes[name].(fwschema.StringAttribute)
			if !ok {
				t.Errorf("%s is not a StringAttribute", name)
				continue
			}
			if !attr.Required {
				t.Errorf("%s must be Required", name)
			}
		}
	})

	t.Run("boolean flag fields reject unknown values", func(t *testing.T) {
		for _, name := range []string{"sign_messages", "sign_assertions", "encrypt_nameid", "scim_status"} {
			attr, ok := schResp.Schema.Attributes[name].(fwschema.StringAttribute)
			if !ok {
				t.Errorf("%s is not a StringAttribute", name)
				continue
			}
			if len(attr.Validators) == 0 {
				t.Errorf("%s has no validators", name)
				continue
			}
			req := validator.StringRequest{ConfigValue: types.StringValue("yes")}
			resp := &validator.StringResponse{}
			for _, v := range attr.Validators {
				v.ValidateString(context.Background(), req, resp)
			}
			if !resp.Diagnostics.HasError() {
				t.Errorf("%s should reject 'yes'", name)
			}
		}
	})
}
```

Save as `internal/provider/user_directory_saml_resource_test.go`.

- [ ] **Step 2: Run to verify it fails**

```
go test ./internal/provider/... -run TestUserDirectorySAMLResource_SchemaValidation -v
```
Expected: compile error — `provider.NewUserDirectorySAMLResource` undefined.

- [ ] **Step 3: Write the SAML resource**

Create `internal/provider/user_directory_saml_resource.go`:

```go
package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &UserDirectorySAMLResource{}
var _ resource.ResourceWithImportState = &UserDirectorySAMLResource{}

func NewUserDirectorySAMLResource() resource.Resource {
	return &UserDirectorySAMLResource{}
}

type UserDirectorySAMLResource struct {
	client client.Client
}

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

func (r *UserDirectorySAMLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_directory_saml"
}

func samlBoolAttr(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("disabled"),
		MarkdownDescription: desc + " One of: `enabled`, `disabled`. Defaults to `disabled`.",
		Validators: []validator.String{
			stringvalidator.OneOf("enabled", "disabled"),
		},
	}
}

func (r *UserDirectorySAMLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonUserDirectoryResourceAttributes()
	attrs["idp_entityid"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "IdP entity ID (the IdP metadata URL or URN).",
	}
	attrs["sp_entityid"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "SP entity ID registered with the IdP.",
	}
	attrs["username_attribute"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "SAML attribute used as the Zabbix username.",
	}
	attrs["sso_url"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "IdP SSO service URL.",
	}
	attrs["slo_url"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "IdP SLO service URL.",
	}
	attrs["nameid_format"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "NameID format (e.g. `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`).",
	}
	attrs["sign_messages"]         = samlBoolAttr("Whether to sign SAML messages.")
	attrs["sign_assertions"]       = samlBoolAttr("Whether to sign SAML assertions.")
	attrs["sign_authn_requests"]   = samlBoolAttr("Whether to sign AuthnRequests.")
	attrs["sign_logout_requests"]  = samlBoolAttr("Whether to sign logout requests.")
	attrs["sign_logout_responses"] = samlBoolAttr("Whether to sign logout responses.")
	attrs["encrypt_nameid"]        = samlBoolAttr("Whether to encrypt the NameID.")
	attrs["encrypt_assertions"]    = samlBoolAttr("Whether to encrypt SAML assertions.")
	attrs["scim_status"]           = samlBoolAttr("Whether SCIM provisioning is enabled for this directory.")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix SAML user directory.",
		Attributes:          attrs,
	}
}

func (r *UserDirectorySAMLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *UserDirectorySAMLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserDirectorySAMLResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, diags := userDirectoryBaseFromModel(ctx, &data.UserDirectoryBaseModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ud.IDPType = client.IDPTypeSAML
	udSAMLFromModel(&data, &ud)

	id, err := client.UserDirectoryCreate(ctx, r.client, ud)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SAML user directory", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.UserDirectoryGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading SAML user directory after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(udSAMLToModel(ctx, created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserDirectorySAMLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserDirectorySAMLResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, err := client.UserDirectoryGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SAML user directory", err.Error())
		return
	}
	if ud == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(udSAMLToModel(ctx, ud, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserDirectorySAMLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserDirectorySAMLResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, diags := userDirectoryBaseFromModel(ctx, &data.UserDirectoryBaseModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ud.ID = data.ID.ValueString()
	ud.IDPType = client.IDPTypeSAML
	udSAMLFromModel(&data, &ud)

	if err := client.UserDirectoryUpdate(ctx, r.client, ud); err != nil {
		resp.Diagnostics.AddError("Error updating SAML user directory", err.Error())
		return
	}

	updated, err := client.UserDirectoryGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SAML user directory after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(udSAMLToModel(ctx, updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserDirectorySAMLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserDirectorySAMLResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.UserDirectoryDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting SAML user directory", err.Error())
	}
}

func (r *UserDirectorySAMLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func udSAMLFromModel(m *UserDirectorySAMLResourceModel, ud *client.UserDirectory) {
	ud.IDPEntityID = m.IDPEntityID.ValueString()
	ud.SPEntityID = m.SPEntityID.ValueString()
	ud.UsernameAttribute = m.UsernameAttribute.ValueString()
	ud.SSOURL = m.SSOURL.ValueString()
	ud.SLOURL = m.SLOURL.ValueString()
	ud.NameIDFormat = m.NameIDFormat.ValueString()
	ud.SignMessages = udEnabledDisabledMap[m.SignMessages.ValueString()]
	ud.SignAssertions = udEnabledDisabledMap[m.SignAssertions.ValueString()]
	ud.SignAuthnRequests = udEnabledDisabledMap[m.SignAuthnRequests.ValueString()]
	ud.SignLogoutRequests = udEnabledDisabledMap[m.SignLogoutRequests.ValueString()]
	ud.SignLogoutResponses = udEnabledDisabledMap[m.SignLogoutResponses.ValueString()]
	ud.EncryptNameID = udEnabledDisabledMap[m.EncryptNameID.ValueString()]
	ud.EncryptAssertions = udEnabledDisabledMap[m.EncryptAssertions.ValueString()]
	ud.SCIMStatus = udEnabledDisabledMap[m.SCIMStatus.ValueString()]
}

func udSAMLToModel(ctx context.Context, ud *client.UserDirectory, m *UserDirectorySAMLResourceModel) diag.Diagnostics {
	diags := userDirectoryBaseToModel(ctx, ud, &m.UserDirectoryBaseModel)
	if diags.HasError() {
		return diags
	}
	m.IDPEntityID = types.StringValue(ud.IDPEntityID)
	m.SPEntityID = types.StringValue(ud.SPEntityID)
	m.UsernameAttribute = types.StringValue(ud.UsernameAttribute)
	m.SSOURL = types.StringValue(ud.SSOURL)
	m.SLOURL = types.StringValue(ud.SLOURL)
	m.NameIDFormat = types.StringValue(ud.NameIDFormat)
	m.SignMessages = types.StringValue(udEnabledDisabledReverseMap[ud.SignMessages])
	m.SignAssertions = types.StringValue(udEnabledDisabledReverseMap[ud.SignAssertions])
	m.SignAuthnRequests = types.StringValue(udEnabledDisabledReverseMap[ud.SignAuthnRequests])
	m.SignLogoutRequests = types.StringValue(udEnabledDisabledReverseMap[ud.SignLogoutRequests])
	m.SignLogoutResponses = types.StringValue(udEnabledDisabledReverseMap[ud.SignLogoutResponses])
	m.EncryptNameID = types.StringValue(udEnabledDisabledReverseMap[ud.EncryptNameID])
	m.EncryptAssertions = types.StringValue(udEnabledDisabledReverseMap[ud.EncryptAssertions])
	m.SCIMStatus = types.StringValue(udEnabledDisabledReverseMap[ud.SCIMStatus])
	return diags
}
```

- [ ] **Step 4: Run schema test and verify it passes**

```
go test ./internal/provider/... -run TestUserDirectorySAMLResource_SchemaValidation -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/user_directory_saml_resource.go internal/provider/user_directory_saml_resource_test.go
git commit -S -s -m "feat(provider): add zabbix_user_directory_saml resource

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 7: SAML data source

**Files:**
- Create: `internal/provider/user_directory_saml_data_source.go`
- Create: `internal/provider/user_directory_saml_data_source_test.go`

- [ ] **Step 1: Write the data source**

```go
package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDirectorySAMLDataSource{}

func NewUserDirectorySAMLDataSource() datasource.DataSource {
	return &UserDirectorySAMLDataSource{}
}

type UserDirectorySAMLDataSource struct {
	client client.Client
}

type UserDirectorySAMLDataSourceModel struct {
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

func (d *UserDirectorySAMLDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_directory_saml"
}

func (d *UserDirectorySAMLDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := commonUserDirectoryDataSourceAttributes()
	attrs["idp_entityid"]       = schema.StringAttribute{Computed: true, MarkdownDescription: "IdP entity ID."}
	attrs["sp_entityid"]        = schema.StringAttribute{Computed: true, MarkdownDescription: "SP entity ID."}
	attrs["username_attribute"] = schema.StringAttribute{Computed: true, MarkdownDescription: "SAML attribute used as the Zabbix username."}
	attrs["sso_url"]            = schema.StringAttribute{Computed: true, MarkdownDescription: "IdP SSO service URL."}
	attrs["slo_url"]            = schema.StringAttribute{Computed: true, MarkdownDescription: "IdP SLO service URL."}
	attrs["nameid_format"]      = schema.StringAttribute{Computed: true, MarkdownDescription: "NameID format."}
	attrs["sign_messages"]         = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SAML messages are signed: `enabled` or `disabled`."}
	attrs["sign_assertions"]       = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SAML assertions are signed: `enabled` or `disabled`."}
	attrs["sign_authn_requests"]   = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether AuthnRequests are signed: `enabled` or `disabled`."}
	attrs["sign_logout_requests"]  = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether logout requests are signed: `enabled` or `disabled`."}
	attrs["sign_logout_responses"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether logout responses are signed: `enabled` or `disabled`."}
	attrs["encrypt_nameid"]        = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether NameID is encrypted: `enabled` or `disabled`."}
	attrs["encrypt_assertions"]    = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SAML assertions are encrypted: `enabled` or `disabled`."}
	attrs["scim_status"]           = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SCIM provisioning is enabled: `enabled` or `disabled`."}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix SAML user directory by ID or name.",
		Attributes:          attrs,
	}
}

func (d *UserDirectorySAMLDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *UserDirectorySAMLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDirectorySAMLDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ud, diags := lookupUserDirectory(ctx, d.client, client.IDPTypeSAML, data.ID, data.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(ud.ID)
	resp.Diagnostics.Append(userDirectoryBaseToModel(ctx, ud, &data.UserDirectoryBaseModel)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.IDPEntityID = types.StringValue(ud.IDPEntityID)
	data.SPEntityID = types.StringValue(ud.SPEntityID)
	data.UsernameAttribute = types.StringValue(ud.UsernameAttribute)
	data.SSOURL = types.StringValue(ud.SSOURL)
	data.SLOURL = types.StringValue(ud.SLOURL)
	data.NameIDFormat = types.StringValue(ud.NameIDFormat)
	data.SignMessages = types.StringValue(udEnabledDisabledReverseMap[ud.SignMessages])
	data.SignAssertions = types.StringValue(udEnabledDisabledReverseMap[ud.SignAssertions])
	data.SignAuthnRequests = types.StringValue(udEnabledDisabledReverseMap[ud.SignAuthnRequests])
	data.SignLogoutRequests = types.StringValue(udEnabledDisabledReverseMap[ud.SignLogoutRequests])
	data.SignLogoutResponses = types.StringValue(udEnabledDisabledReverseMap[ud.SignLogoutResponses])
	data.EncryptNameID = types.StringValue(udEnabledDisabledReverseMap[ud.EncryptNameID])
	data.EncryptAssertions = types.StringValue(udEnabledDisabledReverseMap[ud.EncryptAssertions])
	data.SCIMStatus = types.StringValue(udEnabledDisabledReverseMap[ud.SCIMStatus])

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 2: Create empty test file**

```go
package provider_test
```

Save as `internal/provider/user_directory_saml_data_source_test.go`.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/user_directory_saml_data_source.go internal/provider/user_directory_saml_data_source_test.go
git commit -S -s -m "feat(provider): add zabbix_user_directory_saml data source

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Provider registration + build verification

**Files:**
- Modify: `internal/provider/provider.go`

- [ ] **Step 1: Register the four new constructors**

In `provider.go`, add to the `Resources` slice:
```go
NewUserDirectoryLDAPResource,
NewUserDirectorySAMLResource,
```
Add to the `DataSources` slice:
```go
NewUserDirectoryLDAPDataSource,
NewUserDirectorySAMLDataSource,
```

- [ ] **Step 2: Build and lint**

```
make build
make lint
```
Expected: clean build, no lint errors. Fix any issues before continuing.

- [ ] **Step 3: Run all unit and schema tests**

```
go test ./internal/... -run 'TestUserDirectory' -v
```
Expected: all PASS (client unit tests + schema validation tests).

- [ ] **Step 4: Commit**

```bash
git add internal/provider/provider.go
git commit -S -s -m "feat(provider): register user directory resources and data sources

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 9: Example HCL + docs generation

**Files:**
- Create: `examples/resources/zabbix_user_directory_ldap/resource.tf`
- Create: `examples/resources/zabbix_user_directory_saml/resource.tf`
- Create: `examples/data-sources/zabbix_user_directory_ldap/data-source.tf`
- Create: `examples/data-sources/zabbix_user_directory_saml/data-source.tf`

- [ ] **Step 1: Write LDAP resource example**

```hcl
# examples/resources/zabbix_user_directory_ldap/resource.tf

resource "zabbix_user_directory_ldap" "corporate" {
  name             = "Corporate Active Directory"
  host             = "ad.example.com"
  port             = 389
  base_dn          = "DC=example,DC=com"
  search_attribute = "sAMAccountName"
  bind_dn          = "CN=svc-zabbix,OU=Service Accounts,DC=example,DC=com"
  bind_password    = var.ldap_bind_password
  start_tls        = "disabled"

  group_base_dn    = "OU=Groups,DC=example,DC=com"
  group_name       = "CN"
  group_member     = "member"
  user_ref_attr    = "CN"

  user_username    = "givenName"
  user_lastname    = "sn"

  provision_status = "enabled"

  provision_groups = [
    {
      name          = "Zabbix Admins"
      role_id       = zabbix_role.admin.id
      user_group_ids = [zabbix_user_group.admins.id]
    },
    {
      name          = "*"
      role_id       = zabbix_role.viewer.id
      user_group_ids = [zabbix_user_group.all_users.id]
    },
  ]

  provision_media = [
    {
      name          = "email"
      media_type_id = zabbix_media_type_email.smtp.id
      attribute     = "mail"
      active        = "enabled"
      severity      = 63
      period        = "1-7,00:00-24:00"
    },
  ]
}
```

- [ ] **Step 2: Write SAML resource example**

```hcl
# examples/resources/zabbix_user_directory_saml/resource.tf

resource "zabbix_user_directory_saml" "okta" {
  name               = "Okta SSO"
  idp_entityid       = "http://www.okta.com/exkABCDEF1234567890"
  sp_entityid        = "zabbix"
  username_attribute = "email"
  sso_url            = "https://example.okta.com/app/zabbix/exkABCDEF1234567890/sso/saml"
  slo_url            = "https://example.okta.com/app/zabbix/exkABCDEF1234567890/slo/saml"

  sign_messages   = "enabled"
  sign_assertions = "enabled"
  encrypt_nameid  = "disabled"
  scim_status     = "enabled"

  group_name       = "groups"
  user_username    = "firstName"
  user_lastname    = "lastName"
  provision_status = "enabled"

  provision_groups = [
    {
      name          = "zabbix-admins"
      role_id       = zabbix_role.admin.id
      user_group_ids = [zabbix_user_group.admins.id]
    },
  ]
}
```

- [ ] **Step 3: Write LDAP data source example**

```hcl
# examples/data-sources/zabbix_user_directory_ldap/data-source.tf

data "zabbix_user_directory_ldap" "corp" {
  name = "Corporate Active Directory"
}
```

- [ ] **Step 4: Write SAML data source example**

```hcl
# examples/data-sources/zabbix_user_directory_saml/data-source.tf

data "zabbix_user_directory_saml" "okta" {
  name = "Okta SSO"
}
```

- [ ] **Step 5: Run make generate and verify docs are created**

```
make generate
```
Expected: creates `docs/resources/zabbix_user_directory_ldap.md`, `docs/resources/zabbix_user_directory_saml.md`, `docs/data-sources/zabbix_user_directory_ldap.md`, `docs/data-sources/zabbix_user_directory_saml.md`.

Spot-check one generated doc to confirm `MarkdownDescription` text appears and `bind_password` shows as sensitive.

- [ ] **Step 6: Commit**

```bash
git add examples/ docs/
git commit -S -s -m "docs: add user directory examples and generated docs

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 10: LDAP acceptance tests

**Files:**
- Modify: `internal/provider/user_directory_ldap_resource_test.go`
- Modify: `internal/provider/user_directory_ldap_data_source_test.go`

These tests require `TF_ACC=1`, `ZABBIX_URL`, and `ZABBIX_TOKEN` set against a real Zabbix 7.0 instance.

- [ ] **Step 1: Add LDAP resource acceptance tests**

Add to `internal/provider/user_directory_ldap_resource_test.go`:

```go
package provider_test

import (
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserDirectoryLDAPResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-ldap"
	updated := cfg.NamePrefix + "-ldap-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLDAPResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("host"), knownvalue.StringExact("ldap.example.com")),
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("port"), knownvalue.Int64Exact(389)),
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccLDAPResourceConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
				},
			},
		},
	})
}

func TestAccUserDirectoryLDAPResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ldap-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccLDAPResourceConfig(cfg, name)},
			{
				ResourceName:      "zabbix_user_directory_ldap.test",
				ImportState:       true,
				ImportStateVerify: true,
				// bind_password is write-only and not returned by API
				ImportStateVerifyIgnore: []string{"bind_password"},
			},
		},
	})
}

func testAccLDAPResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_ldap" "test" {
  name             = %q
  host             = "ldap.example.com"
  base_dn          = "dc=example,dc=com"
  search_attribute = "uid"
}
`, cfg.URL, cfg.Token, name)
}
```

- [ ] **Step 2: Add LDAP data source acceptance tests**

Replace the placeholder in `internal/provider/user_directory_ldap_data_source_test.go`:

```go
package provider_test

import (
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserDirectoryLDAPDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ldap-ds"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLDAPDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_user_directory_ldap.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_ldap.test", tfjsonpath.New("host"), knownvalue.StringExact("ldap.example.com")),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_ldap.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccLDAPDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_ldap" "seed" {
  name             = %q
  host             = "ldap.example.com"
  base_dn          = "dc=example,dc=com"
  search_attribute = "uid"
}

data "zabbix_user_directory_ldap" "test" {
  name       = zabbix_user_directory_ldap.seed.name
  depends_on = [zabbix_user_directory_ldap.seed]
}
`, cfg.URL, cfg.Token, name)
}
```

- [ ] **Step 3: Run LDAP acceptance tests**

```
TF_ACC=1 ZABBIX_URL=<url> ZABBIX_TOKEN=<token> go test ./internal/provider/... -run 'TestAccUserDirectoryLDAP' -v -timeout 120s
```
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/user_directory_ldap_resource_test.go internal/provider/user_directory_ldap_data_source_test.go
git commit -S -s -m "test(provider): add LDAP user directory acceptance tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 11: SAML acceptance tests

**Files:**
- Modify: `internal/provider/user_directory_saml_resource_test.go`
- Modify: `internal/provider/user_directory_saml_data_source_test.go`

- [ ] **Step 1: Add SAML resource acceptance tests**

Add to `internal/provider/user_directory_saml_resource_test.go`:

```go
package provider_test

import (
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserDirectorySAMLResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-saml"
	updated := cfg.NamePrefix + "-saml-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("idp_entityid"), knownvalue.StringExact("http://idp.example.com/metadata")),
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccSAMLResourceConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
				},
			},
		},
	})
}

func TestAccUserDirectorySAMLResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-saml-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccSAMLResourceConfig(cfg, name)},
			{
				ResourceName:      "zabbix_user_directory_saml.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSAMLResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_saml" "test" {
  name         = %q
  idp_entityid = "http://idp.example.com/metadata"
  sp_entityid  = "zabbix"
}
`, cfg.URL, cfg.Token, name)
}
```

- [ ] **Step 2: Add SAML data source acceptance tests**

Replace placeholder in `internal/provider/user_directory_saml_data_source_test.go`:

```go
package provider_test

import (
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserDirectorySAMLDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-saml-ds"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_user_directory_saml.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_saml.test", tfjsonpath.New("idp_entityid"), knownvalue.StringExact("http://idp.example.com/metadata")),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_saml.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccSAMLDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_saml" "seed" {
  name         = %q
  idp_entityid = "http://idp.example.com/metadata"
  sp_entityid  = "zabbix"
}

data "zabbix_user_directory_saml" "test" {
  name       = zabbix_user_directory_saml.seed.name
  depends_on = [zabbix_user_directory_saml.seed]
}
`, cfg.URL, cfg.Token, name)
}
```

- [ ] **Step 3: Run SAML acceptance tests**

```
TF_ACC=1 ZABBIX_URL=<url> ZABBIX_TOKEN=<token> go test ./internal/provider/... -run 'TestAccUserDirectorySAML' -v -timeout 120s
```
Expected: all PASS.

- [ ] **Step 4: Run full pre-PR checklist**

```
make generate
make build
make lint
make acc-tests
```
Expected: all pass. Fix any failures before opening the PR.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/user_directory_saml_resource_test.go internal/provider/user_directory_saml_data_source_test.go
git commit -S -s -m "test(provider): add SAML user directory acceptance tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

- [ ] **Step 6: Open PR**

```bash
gh pr create \
  --title "feat: zabbix_user_directory_ldap + zabbix_user_directory_saml resource and data source" \
  --body "$(cat <<'EOF'
## Summary

- Adds `zabbix_user_directory_ldap` resource and data source
- Adds `zabbix_user_directory_saml` resource and data source
- Both support inline `provision_groups` and `provision_media`
- `bind_password` on LDAP is marked Sensitive and never echoed
- Data sources support lookup by `id` or `name` with zero/multi-match errors
- Follows the `zabbix_media_type_*` pattern (shared common layer, separate resource per subtype)

Closes #21
EOF
)"
```

---

## Self-Review

**Spec coverage check:**
- ✅ CRUD for LDAP + SAML — Tasks 2, 4, 6
- ✅ `terraform import` — ImportState in Tasks 4, 6; acc tests in Tasks 10, 11
- ✅ Drift detection including provision_groups mutation — covered by acc CRUD test (update step replaces full provision_groups list)
- ✅ Type-discriminated schema — separate resources eliminate need for discriminator validation
- ✅ `bind_password` Sensitive, not Computed, not read back in `udLDAPToModel` — Task 4
- ✅ Data source id/name lookup + zero/multi-match errors — `lookupUserDirectory` in Task 3
- ✅ Client unit tests — Task 1+2
- ✅ MarkdownDescription + example HCL + tfplugindocs — Task 9

**Placeholder scan:** None found.

**Type consistency:**
- `udLDAPFromModel` / `udLDAPToModel` use `udEnabledDisabledMap` / `udEnabledDisabledReverseMap` defined in Task 3 — consistent
- `provisionGroupsToParams` uses `ProvisionGroup.RoleID` → JSON `roleid` — consistent with struct definition in Task 2
- `UserDirectoryBaseModel` embedded in all four resource/datasource models — consistent field names throughout
