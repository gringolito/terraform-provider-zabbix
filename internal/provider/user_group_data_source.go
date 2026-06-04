package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserGroupDataSource{}

func NewUserGroupDataSource() datasource.DataSource {
	return &UserGroupDataSource{}
}

type UserGroupDataSource struct {
	client client.Client
}

type UserGroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	GUIAccess   types.String `tfsdk:"gui_access"`
	DebugMode   types.String `tfsdk:"debug_mode"`
	UsersStatus types.String `tfsdk:"users_status"`
}

func (d *UserGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (d *UserGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix user group by ID or name. Exactly one of `id` or `name` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the user group. One of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display name of the user group. One of `id` or `name` must be set.",
			},
			"gui_access": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Frontend authentication method: `system_default`, `internal`, or `disabled`.",
			},
			"debug_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Debug mode for the group: `disabled` or `enabled`.",
			},
			"users_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Status of the users in this group: `enabled` or `disabled`.",
			},
		},
	}
}

func (d *UserGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Exactly one of `id` or `name` must be set.",
		)
		return
	}

	if !data.ID.IsNull() {
		ug, err := client.UserGroupGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user group", err.Error())
			return
		}
		if ug == nil {
			resp.Diagnostics.AddError(
				"User group not found",
				fmt.Sprintf("No user group found with id %q.", data.ID.ValueString()),
			)
			return
		}
		data.ID = types.StringValue(ug.ID)
		data.Name = types.StringValue(ug.Name)
		data.GUIAccess = types.StringValue(guiAccessReverseMap[ug.GUIAccess])
		data.DebugMode = types.StringValue(debugModeReverseMap[ug.DebugMode])
		data.UsersStatus = types.StringValue(usersStatusReverseMap[ug.UsersStatus])
	} else {
		groups, err := client.UserGroupGetByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user group", err.Error())
			return
		}
		switch len(groups) {
		case 0:
			resp.Diagnostics.AddError(
				"User group not found",
				fmt.Sprintf("No user group found with name %q.", data.Name.ValueString()),
			)
			return
		case 1:
			data.ID = types.StringValue(groups[0].ID)
			data.Name = types.StringValue(groups[0].Name)
			data.GUIAccess = types.StringValue(guiAccessReverseMap[groups[0].GUIAccess])
			data.DebugMode = types.StringValue(debugModeReverseMap[groups[0].DebugMode])
			data.UsersStatus = types.StringValue(usersStatusReverseMap[groups[0].UsersStatus])
		default:
			resp.Diagnostics.AddError(
				"Multiple user groups found",
				fmt.Sprintf("Found %d user groups with name %q; use `id` to disambiguate.", len(groups), data.Name.ValueString()),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
