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
	attrs["host"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Hostname or IP address of the LDAP server."}
	attrs["port"] = schema.Int64Attribute{Computed: true, MarkdownDescription: "Port of the LDAP server."}
	attrs["base_dn"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Base DN for LDAP search."}
	attrs["search_attribute"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute used to identify users."}
	attrs["bind_dn"] = schema.StringAttribute{Computed: true, MarkdownDescription: "DN used to bind to the LDAP server."}
	attrs["start_tls"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Whether StartTLS is enabled: `enabled` or `disabled`."}
	attrs["search_filter"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Custom LDAP search filter."}
	attrs["group_base_dn"] = schema.StringAttribute{Computed: true, MarkdownDescription: "Base DN for group search."}
	attrs["group_member"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute listing group members."}
	attrs["group_filter"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP filter for group search."}
	attrs["group_membership"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute on user listing their groups."}
	attrs["user_ref_attr"] = schema.StringAttribute{Computed: true, MarkdownDescription: "LDAP attribute referencing users in group entries."}
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
