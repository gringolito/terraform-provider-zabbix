package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AuthenticationDataSource{}

func NewAuthenticationDataSource() datasource.DataSource {
	return &AuthenticationDataSource{}
}

type AuthenticationDataSource struct {
	client client.Client
}

type AuthenticationDataSourceModel struct {
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
	PasswdMinLength      types.Int64  `tfsdk:"passwd_min_length"`
	PasswdCheckRules     types.Set    `tfsdk:"passwd_check_rules"`
	JITProvisionInterval types.String `tfsdk:"jit_provision_interval"`
	SAMLJITStatus        types.String `tfsdk:"saml_jit_status"`
	LDAPJITStatus        types.String `tfsdk:"ldap_jit_status"`
	DisabledUsrgrpID     types.String `tfsdk:"disabled_usrgrpid"`
	MFAStatus            types.String `tfsdk:"mfa_status"`
	MFAID                types.String `tfsdk:"mfaid"`
}

func (d *AuthenticationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authentication"
}

func (d *AuthenticationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Zabbix global authentication configuration singleton.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthesized constant identifier. Always `\"authentication\"`.",
			},
			"authentication_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Default authentication method. One of: `internal`, `ldap`.",
			},
			"http_auth_enabled": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether HTTP authentication is enabled. One of: `enabled`, `disabled`.",
			},
			"http_login_form": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Login form shown when HTTP auth is active. One of: `zabbix_login_form`, `http_login_form`.",
			},
			"http_strip_domains": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Comma-separated domain names to strip from HTTP authentication usernames.",
			},
			"http_case_sensitive": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether HTTP authentication is case-sensitive. One of: `enabled`, `disabled`.",
			},
			"ldap_auth_enabled": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether LDAP authentication is enabled. One of: `enabled`, `disabled`.",
			},
			"ldap_case_sensitive": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether LDAP authentication is case-sensitive. One of: `enabled`, `disabled`.",
			},
			"ldap_userdirectoryid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the default LDAP user directory.",
			},
			"saml_auth_enabled": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether SAML authentication is enabled. One of: `enabled`, `disabled`.",
			},
			"saml_case_sensitive": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether SAML authentication is case-sensitive. One of: `enabled`, `disabled`.",
			},
			"passwd_min_length": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Minimum password length (1–70).",
			},
			"passwd_check_rules": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Password complexity rules enforced. Values: `case_sensitive_letters`, `digits`, `special_characters`, `avoid_common_passwords`.",
			},
			"jit_provision_interval": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Interval between JIT provisioning requests (e.g. `1h`, `60m`).",
			},
			"saml_jit_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether SAML JIT provisioning is enabled. One of: `enabled`, `disabled`.",
			},
			"ldap_jit_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether LDAP JIT provisioning is enabled. One of: `enabled`, `disabled`.",
			},
			"disabled_usrgrpid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the user group for deprovisioned users.",
			},
			"mfa_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether multi-factor authentication is enabled. One of: `enabled`, `disabled`.",
			},
			"mfaid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the default MFA method. Returns `\"0\"` when no MFA factor is configured.",
			},
		},
	}
}

func (d *AuthenticationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AuthenticationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AuthenticationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	auth, err := client.AuthenticationGet(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error reading authentication", err.Error())
		return
	}

	data.ID = types.StringValue("authentication")
	resp.Diagnostics.Append(authDSToModel(auth, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func authDSToModel(auth *client.Authentication, m *AuthenticationDataSourceModel) diag.Diagnostics {
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
	m.PasswdMinLength = types.Int64Value(auth.PasswdMinLength)
	m.JITProvisionInterval = types.StringValue(auth.JITProvisionInterval)
	m.SAMLJITStatus = types.StringValue(authEnabledDisabledReverseMap[auth.SAMLJITStatus])
	m.LDAPJITStatus = types.StringValue(authEnabledDisabledReverseMap[auth.LDAPJITStatus])
	m.DisabledUsrgrpID = types.StringValue(auth.DisabledUsrgrpID)
	m.MFAStatus = types.StringValue(authEnabledDisabledReverseMap[auth.MFAStatus])
	m.MFAID = types.StringValue(auth.MFAID)

	ruleNames := passwdRulesFromBitmask(auth.PasswdCheckRules)
	vals := make([]attr.Value, len(ruleNames))
	for i, name := range ruleNames {
		vals[i] = types.StringValue(name)
	}
	var d diag.Diagnostics
	m.PasswdCheckRules, d = types.SetValue(types.StringType, vals)
	diags.Append(d...)

	return diags
}
