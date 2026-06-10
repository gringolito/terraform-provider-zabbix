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
	attrs["sign_messages"] = samlBoolAttr("Whether to sign SAML messages.")
	attrs["sign_assertions"] = samlBoolAttr("Whether to sign SAML assertions.")
	attrs["sign_authn_requests"] = samlBoolAttr("Whether to sign AuthnRequests.")
	attrs["sign_logout_requests"] = samlBoolAttr("Whether to sign logout requests.")
	attrs["sign_logout_responses"] = samlBoolAttr("Whether to sign logout responses.")
	attrs["encrypt_nameid"] = samlBoolAttr("Whether to encrypt the NameID.")
	attrs["encrypt_assertions"] = samlBoolAttr("Whether to encrypt SAML assertions.")
	attrs["scim_status"] = samlBoolAttr("Whether SCIM provisioning is enabled for this directory.")
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
