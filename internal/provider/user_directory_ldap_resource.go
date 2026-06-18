package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
		MarkdownDescription: "LDAP server port (1–65535). Defaults to `389`.",
		Validators: []validator.Int64{
			int64validator.Between(1, 65535),
		},
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
	// m.BindPassword intentionally not updated — write-only
	m.StartTLS = types.StringValue(udEnabledDisabledReverseMap[ud.StartTLS])
	m.SearchFilter = types.StringValue(ud.SearchFilter)
	m.GroupBaseDN = types.StringValue(ud.GroupBaseDN)
	m.GroupMember = types.StringValue(ud.GroupMember)
	m.GroupFilter = types.StringValue(ud.GroupFilter)
	m.GroupMembership = types.StringValue(ud.GroupMembership)
	m.UserRefAttr = types.StringValue(ud.UserRefAttr)
	return diags
}
