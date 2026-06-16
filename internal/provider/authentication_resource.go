package provider

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AuthenticationResource{}
var _ resource.ResourceWithImportState = &AuthenticationResource{}

// passwd_check_rules bitmask — Zabbix 7.0 API reference:
// https://www.zabbix.com/documentation/7.0/en/manual/api/reference/authentication/object
var passwdCheckRulesMap = map[string]int64{
	"case_sensitive_letters": 1,
	"digits":                 2,
	"special_characters":     4,
	"avoid_common_passwords": 8,
}

var passwdCheckRulesReverseMap = map[int64]string{
	1: "case_sensitive_letters",
	2: "digits",
	4: "special_characters",
	8: "avoid_common_passwords",
}

var (
	authTypeMap = map[string]int64{
		"internal": 0,
		"ldap":     1,
	}
	authTypeReverseMap = map[int64]string{
		0: "internal",
		1: "ldap",
	}
	authEnabledDisabledMap = map[string]int64{
		"disabled": 0,
		"enabled":  1,
	}
	authEnabledDisabledReverseMap = map[int64]string{
		0: "disabled",
		1: "enabled",
	}
	httpLoginFormMap = map[string]int64{
		"zabbix_login_form": 0,
		"http_login_form":   1,
	}
	httpLoginFormReverseMap = map[int64]string{
		0: "zabbix_login_form",
		1: "http_login_form",
	}
)

var (
	authSingletonMu  sync.Mutex
	authSingletonSet = make(map[client.Client]struct{})
)

func NewAuthenticationResource() resource.Resource {
	return &AuthenticationResource{}
}

type AuthenticationResource struct {
	client client.Client
}

type AuthenticationResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	AuthenticationType   types.String `tfsdk:"authentication_type"`
	HTTPAuthEnabled      types.String `tfsdk:"http_auth_enabled"`
	HTTPLoginForm        types.String `tfsdk:"http_login_form"`
	HTTPStripDomains     types.String `tfsdk:"http_strip_domains"`
	HTTPCaseSensitive    types.String `tfsdk:"http_case_sensitive"`
	LDAPAuthEnabled      types.String `tfsdk:"ldap_auth_enabled"`
	LDAPCaseSensitive    types.String `tfsdk:"ldap_case_sensitive"`
	LDAPUserDirectoryID  types.String `tfsdk:"ldap_userdirectoryid"`
	SAMLAuthEnabled      types.String `tfsdk:"saml_auth_enabled"`
	SAMLCaseSensitive    types.String `tfsdk:"saml_case_sensitive"`
	PasswordMinLength    types.Int64  `tfsdk:"password_min_length"`
	PasswordCheckRules   types.Set    `tfsdk:"password_check_rules"`
	JITProvisionInterval types.String `tfsdk:"jit_provision_interval"`
	SAMLJITStatus        types.String `tfsdk:"saml_jit_status"`
	LDAPJITStatus        types.String `tfsdk:"ldap_jit_status"`
	DisabledUsergroupID  types.String `tfsdk:"disabled_usergroupid"`
	MFAStatus            types.String `tfsdk:"mfa_status"`
	MFAID                types.String `tfsdk:"mfaid"`
}

func (r *AuthenticationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authentication"
}

func (r *AuthenticationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages the Zabbix global authentication configuration.

**This is a singleton resource.** Zabbix has exactly one authentication object — it cannot be
created or deleted, only updated. The provider adopts it on ` + "`terraform apply`" + ` and resets
it to Zabbix 7.0 documented defaults on ` + "`terraform destroy`" + `.

**Warning:** Declaring ` + "`zabbix_authentication`" + ` more than once in the same Terraform
configuration is a footgun — both blocks will fight over the same singleton and produce
non-deterministic results. Always declare it exactly once.

To import the existing singleton:

` + "```" + `
terraform import zabbix_authentication.example authentication
` + "```",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthesized constant identifier. Always `\"authentication\"`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"authentication_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("internal"),
				MarkdownDescription: "Default authentication method. One of: `internal`, `ldap`. Defaults to `internal`.",
				Validators: []validator.String{
					stringvalidator.OneOf("internal", "ldap"),
				},
			},
			"http_auth_enabled": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Whether HTTP authentication is enabled. One of: `enabled`, `disabled`. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"http_login_form": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("zabbix_login_form"),
				MarkdownDescription: "Login form shown when HTTP auth is active. One of: `zabbix_login_form`, `http_login_form`. Defaults to `zabbix_login_form`.",
				Validators: []validator.String{
					stringvalidator.OneOf("zabbix_login_form", "http_login_form"),
				},
			},
			"http_strip_domains": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Comma-separated domain names to strip from HTTP authentication usernames.",
			},
			"http_case_sensitive": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Whether HTTP authentication is case-sensitive. One of: `enabled`, `disabled`. Defaults to `enabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"ldap_auth_enabled": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Whether LDAP authentication is enabled. One of: `enabled`, `disabled`. Requires `ldap_userdirectoryid` when set. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
					stringvalidator.AlsoRequires(path.MatchRoot("ldap_userdirectoryid")),
				},
			},
			"ldap_case_sensitive": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Whether LDAP authentication is case-sensitive. One of: `enabled`, `disabled`. Defaults to `enabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"ldap_userdirectoryid": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0"),
				MarkdownDescription: "ID of the default LDAP user directory. Required when `ldap_auth_enabled = \"enabled\"`. Defaults to `\"0\"` (none).",
			},
			"saml_auth_enabled": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Whether SAML authentication is enabled. One of: `enabled`, `disabled`. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"saml_case_sensitive": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Whether SAML authentication is case-sensitive. One of: `enabled`, `disabled`. Defaults to `enabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"password_min_length": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(8),
				MarkdownDescription: "Minimum password length (1–70). Defaults to `8`.",
				Validators: []validator.Int64{
					int64validator.Between(1, 70),
				},
			},
			"password_check_rules": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Password complexity rules to enforce. Valid values: `case_sensitive_letters`, `digits`, `special_characters`, `avoid_common_passwords`. Defaults to all four rules when not specified.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf("case_sensitive_letters", "digits", "special_characters", "avoid_common_passwords"),
					),
				},
			},
			"jit_provision_interval": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1h"),
				MarkdownDescription: "Interval between JIT provisioning requests (e.g. `1h`, `60m`). Minimum `1h`. Defaults to `1h`.",
			},
			"saml_jit_status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Whether SAML JIT provisioning is enabled. One of: `enabled`, `disabled`. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"ldap_jit_status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Whether LDAP JIT provisioning is enabled. One of: `enabled`, `disabled`. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"disabled_usergroupid": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the user group for deprovisioned users. Required when JIT provisioning is enabled for LDAP or SAML.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mfa_status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Whether multi-factor authentication is enabled. One of: `enabled`, `disabled`. Requires `mfaid` when set. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
					stringvalidator.AlsoRequires(path.MatchRoot("mfaid")),
				},
			},
			"mfaid": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0"),
				MarkdownDescription: "ID of the default MFA method. Required when `mfa_status = \"enabled\"`. Defaults to `\"0\"` (none).",
			},
		},
	}
}

func (r *AuthenticationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create adopts the existing Zabbix authentication singleton and applies the desired config.
// The singleton always exists — Create never calls authentication.create.
// A read-before-write ensures optional+computed fields the user did not specify
// receive their current Zabbix values rather than empty strings.
func (r *AuthenticationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	authSingletonMu.Lock()
	if _, exists := authSingletonSet[r.client]; exists {
		authSingletonMu.Unlock()
		resp.Diagnostics.AddError(
			"Duplicate zabbix_authentication resource",
			"Only one zabbix_authentication resource may be declared per Terraform configuration. "+
				"Multiple blocks fight over the same Zabbix singleton and produce non-deterministic results.",
		)
		return
	}
	authSingletonSet[r.client] = struct{}{}
	authSingletonMu.Unlock()

	var data AuthenticationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the current singleton state so we can fill in any optional+computed
	// fields the user left unset (they would be Unknown/null in the plan).
	current, err := client.AuthenticationGet(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError("Error reading authentication for adoption", err.Error())
		return
	}
	authFillUnknownFromCurrent(ctx, &data, current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	auth, diags := authFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.AuthenticationUpdate(ctx, r.client, auth); err != nil {
		resp.Diagnostics.AddError("Error updating authentication", err.Error())
		return
	}

	data.ID = types.StringValue("authentication")

	updated, err := client.AuthenticationGet(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError("Error reading authentication after create", err.Error())
		return
	}
	resp.Diagnostics.Append(authToModel(ctx, updated, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AuthenticationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AuthenticationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	auth, err := client.AuthenticationGet(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError("Error reading authentication", err.Error())
		return
	}
	resp.Diagnostics.Append(authToModel(ctx, auth, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AuthenticationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AuthenticationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	auth, diags := authFromModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.AuthenticationUpdate(ctx, r.client, auth); err != nil {
		resp.Diagnostics.AddError("Error updating authentication", err.Error())
		return
	}

	updated, err := client.AuthenticationGet(ctx, r.client)
	if err != nil {
		resp.Diagnostics.AddError("Error reading authentication after update", err.Error())
		return
	}
	resp.Diagnostics.Append(authToModel(ctx, updated, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete resets the authentication singleton to Zabbix 7.0 documented defaults.
// The object cannot be truly deleted; this emits a diagnostic warning per ADR-0014.
// Defaults sourced from: https://www.zabbix.com/documentation/7.0/en/manual/api/reference/authentication/object
func (r *AuthenticationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonWarnOnDelete(ctx, &resp.Diagnostics)

	defaults := client.Authentication{
		AuthenticationType:   0, // internal
		HTTPAuthEnabled:      0, // disabled
		HTTPLoginForm:        0, // zabbix_login_form
		HTTPStripDomains:     "",
		HTTPCaseSensitive:    1, // enabled
		LDAPAuthEnabled:      0, // disabled
		LDAPCaseSensitive:    1, // enabled
		LDAPUserDirectoryID:  "0",
		SAMLAuthEnabled:      0, // disabled
		SAMLCaseSensitive:    1, // enabled
		PasswdMinLength:      8,
		PasswdCheckRules:     15, // all four rules: case_sensitive_letters|digits|special_characters|avoid_common_passwords
		JITProvisionInterval: "1h",
		SAMLJITStatus:        0,   // disabled
		LDAPJITStatus:        0,   // disabled
		DisabledUsrgrpID:     "9", // built-in "Disabled" user group in a default Zabbix 7.0 installation
		MFAStatus:            0,   // disabled
		MFAID:                "0",
	}
	if err := client.AuthenticationUpdate(ctx, r.client, defaults); err != nil {
		resp.Diagnostics.AddError("Error resetting authentication to defaults", err.Error())
	}

	authSingletonMu.Lock()
	delete(authSingletonSet, r.client)
	authSingletonMu.Unlock()
}

func (r *AuthenticationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// authFillUnknownFromCurrent replaces null/unknown optional+computed fields in
// the model with values from the current Zabbix state. Required for Create
// (adopt) so that unset fields receive valid values rather than empty strings.
func authFillUnknownFromCurrent(_ context.Context, m *AuthenticationResourceModel, cur *client.Authentication, diags *diag.Diagnostics) {
	if m.DisabledUsergroupID.IsNull() || m.DisabledUsergroupID.IsUnknown() {
		m.DisabledUsergroupID = types.StringValue(cur.DisabledUsrgrpID)
	}
	if m.PasswordCheckRules.IsNull() || m.PasswordCheckRules.IsUnknown() {
		ruleNames := passwdRulesFromBitmask(cur.PasswdCheckRules)
		vals := make([]attr.Value, len(ruleNames))
		for i, name := range ruleNames {
			vals[i] = types.StringValue(name)
		}
		var d diag.Diagnostics
		m.PasswordCheckRules, d = types.SetValue(types.StringType, vals)
		diags.Append(d...)
	}
}

func authFromModel(ctx context.Context, m *AuthenticationResourceModel) (client.Authentication, diag.Diagnostics) {
	var diags diag.Diagnostics

	var rules []string
	if !m.PasswordCheckRules.IsNull() && !m.PasswordCheckRules.IsUnknown() {
		diags.Append(m.PasswordCheckRules.ElementsAs(ctx, &rules, false)...)
		if diags.HasError() {
			return client.Authentication{}, diags
		}
	}

	return client.Authentication{
		AuthenticationType:   authTypeMap[m.AuthenticationType.ValueString()],
		HTTPAuthEnabled:      authEnabledDisabledMap[m.HTTPAuthEnabled.ValueString()],
		HTTPLoginForm:        httpLoginFormMap[m.HTTPLoginForm.ValueString()],
		HTTPStripDomains:     m.HTTPStripDomains.ValueString(),
		HTTPCaseSensitive:    authEnabledDisabledMap[m.HTTPCaseSensitive.ValueString()],
		LDAPAuthEnabled:      authEnabledDisabledMap[m.LDAPAuthEnabled.ValueString()],
		LDAPCaseSensitive:    authEnabledDisabledMap[m.LDAPCaseSensitive.ValueString()],
		LDAPUserDirectoryID:  m.LDAPUserDirectoryID.ValueString(),
		SAMLAuthEnabled:      authEnabledDisabledMap[m.SAMLAuthEnabled.ValueString()],
		SAMLCaseSensitive:    authEnabledDisabledMap[m.SAMLCaseSensitive.ValueString()],
		PasswdMinLength:      m.PasswordMinLength.ValueInt64(),
		PasswdCheckRules:     passwdRulesToBitmask(rules),
		JITProvisionInterval: m.JITProvisionInterval.ValueString(),
		SAMLJITStatus:        authEnabledDisabledMap[m.SAMLJITStatus.ValueString()],
		LDAPJITStatus:        authEnabledDisabledMap[m.LDAPJITStatus.ValueString()],
		DisabledUsrgrpID:     m.DisabledUsergroupID.ValueString(),
		MFAStatus:            authEnabledDisabledMap[m.MFAStatus.ValueString()],
		MFAID:                m.MFAID.ValueString(),
	}, diags
}

func authToModel(_ context.Context, auth *client.Authentication, m *AuthenticationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.AuthenticationType = types.StringValue(authTypeReverseMap[auth.AuthenticationType])
	m.HTTPAuthEnabled = types.StringValue(authEnabledDisabledReverseMap[auth.HTTPAuthEnabled])
	m.HTTPLoginForm = types.StringValue(httpLoginFormReverseMap[auth.HTTPLoginForm])
	m.HTTPStripDomains = types.StringValue(auth.HTTPStripDomains)
	m.HTTPCaseSensitive = types.StringValue(authEnabledDisabledReverseMap[auth.HTTPCaseSensitive])
	m.LDAPAuthEnabled = types.StringValue(authEnabledDisabledReverseMap[auth.LDAPAuthEnabled])
	m.LDAPCaseSensitive = types.StringValue(authEnabledDisabledReverseMap[auth.LDAPCaseSensitive])
	m.LDAPUserDirectoryID = types.StringValue(auth.LDAPUserDirectoryID)
	m.SAMLAuthEnabled = types.StringValue(authEnabledDisabledReverseMap[auth.SAMLAuthEnabled])
	m.SAMLCaseSensitive = types.StringValue(authEnabledDisabledReverseMap[auth.SAMLCaseSensitive])
	m.PasswordMinLength = types.Int64Value(auth.PasswdMinLength)
	m.JITProvisionInterval = types.StringValue(auth.JITProvisionInterval)
	m.SAMLJITStatus = types.StringValue(authEnabledDisabledReverseMap[auth.SAMLJITStatus])
	m.LDAPJITStatus = types.StringValue(authEnabledDisabledReverseMap[auth.LDAPJITStatus])
	m.DisabledUsergroupID = types.StringValue(auth.DisabledUsrgrpID)
	m.MFAStatus = types.StringValue(authEnabledDisabledReverseMap[auth.MFAStatus])
	m.MFAID = types.StringValue(auth.MFAID)

	ruleNames := passwdRulesFromBitmask(auth.PasswdCheckRules)
	vals := make([]attr.Value, len(ruleNames))
	for i, name := range ruleNames {
		vals[i] = types.StringValue(name)
	}
	var d diag.Diagnostics
	m.PasswordCheckRules, d = types.SetValue(types.StringType, vals)
	diags.Append(d...)

	return diags
}

func passwdRulesToBitmask(rules []string) int64 {
	var bitmask int64
	for _, r := range rules {
		bitmask |= passwdCheckRulesMap[r]
	}
	return bitmask
}

func passwdRulesFromBitmask(bitmask int64) []string {
	var rules []string
	for _, bit := range []int64{1, 2, 4, 8} {
		if bitmask&bit != 0 {
			rules = append(rules, passwdCheckRulesReverseMap[bit])
		}
	}
	sort.Strings(rules)
	return rules
}
