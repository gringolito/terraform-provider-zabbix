package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

type UserDataSource struct {
	client client.Client
}

type UserDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Name     types.String `tfsdk:"name"`
	Surname  types.String `tfsdk:"surname"`
	Type     types.String `tfsdk:"type"`
	RoleID   types.String `tfsdk:"role_id"`
}

var userTypeReverseMap = map[int64]string{
	1: "user", 2: "admin", 3: "super_admin",
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix user by `id` or `username`. Exactly one of `id` or `username` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the user. One of `id` or `username` must be set.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Login name of the user. Unique per Zabbix instance. One of `id` or `username` must be set.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "First name of the user.",
			},
			"surname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last name of the user.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User permission level: `user`, `admin`, or `super_admin`.",
			},
			"role_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the role assigned to the user.",
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Username.IsNull() {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Exactly one of `id` or `username` must be set.",
		)
		return
	}

	if !data.ID.IsNull() {
		user, err := client.UserGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", err.Error())
			return
		}
		if user == nil {
			resp.Diagnostics.AddError(
				"User not found",
				fmt.Sprintf("No user found with id %q.", data.ID.ValueString()),
			)
			return
		}
		data.ID = types.StringValue(user.UserID)
		data.Username = types.StringValue(user.Username)
		data.Name = types.StringValue(user.Name)
		data.Surname = types.StringValue(user.Surname)
		data.Type = types.StringValue(userTypeReverseMap[user.Type])
		data.RoleID = types.StringValue(user.RoleID)
	} else {
		users, err := client.UserGetByUsername(ctx, d.client, data.Username.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", err.Error())
			return
		}
		switch len(users) {
		case 0:
			resp.Diagnostics.AddError(
				"User not found",
				fmt.Sprintf("No user found with username %q.", data.Username.ValueString()),
			)
			return
		case 1:
			data.ID = types.StringValue(users[0].UserID)
			data.Username = types.StringValue(users[0].Username)
			data.Name = types.StringValue(users[0].Name)
			data.Surname = types.StringValue(users[0].Surname)
			data.Type = types.StringValue(userTypeReverseMap[users[0].Type])
			data.RoleID = types.StringValue(users[0].RoleID)
		default:
			resp.Diagnostics.AddError(
				"Multiple users found",
				fmt.Sprintf("Found %d users with username %q; use `id` to disambiguate.", len(users), data.Username.ValueString()),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
