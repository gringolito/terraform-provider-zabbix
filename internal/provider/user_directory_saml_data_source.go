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
	attrs["idp_entityid"] = schema.StringAttribute{Computed: true, MarkdownDescription: "IdP entity ID."}
	attrs["sp_entityid"] = schema.StringAttribute{Computed: true, MarkdownDescription: "SP entity ID."}
	attrs["username_attribute"] = schema.StringAttribute{Computed: true, MarkdownDescription: "SAML attribute used as the Zabbix username."}
	attrs["sso_url"] = schema.StringAttribute{Computed: true, MarkdownDescription: "IdP SSO service URL."}
	attrs["slo_url"] = schema.StringAttribute{Computed: true, MarkdownDescription: "IdP SLO service URL."}
	attrs["nameid_format"] = schema.StringAttribute{Computed: true, MarkdownDescription: "NameID format."}
	attrs["sign_messages"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SAML messages are signed: `enabled` or `disabled`."}
	attrs["sign_assertions"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SAML assertions are signed: `enabled` or `disabled`."}
	attrs["sign_authn_requests"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether AuthnRequests are signed: `enabled` or `disabled`."}
	attrs["sign_logout_requests"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether logout requests are signed: `enabled` or `disabled`."}
	attrs["sign_logout_responses"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether logout responses are signed: `enabled` or `disabled`."}
	attrs["encrypt_nameid"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether NameID is encrypted: `enabled` or `disabled`."}
	attrs["encrypt_assertions"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SAML assertions are encrypted: `enabled` or `disabled`."}
	attrs["scim_status"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether SCIM provisioning is enabled: `enabled` or `disabled`."}
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
