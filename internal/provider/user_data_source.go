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
	ID            types.String `tfsdk:"id"`
	Username      types.String `tfsdk:"username"`
	Name          types.String `tfsdk:"name"`
	Surname       types.String `tfsdk:"surname"`
	URL           types.String `tfsdk:"url"`
	AutoLogin     types.Bool   `tfsdk:"auto_login"`
	AutoLogout    types.String `tfsdk:"auto_logout"`
	Language      types.String `tfsdk:"language"`
	Refresh       types.String `tfsdk:"refresh"`
	Theme         types.String `tfsdk:"theme"`
	AttemptFailed types.String `tfsdk:"attempt_failed"`
	AttemptIP     types.String `tfsdk:"attempt_ip"`
	AttemptClock  types.String `tfsdk:"attempt_clock"`
	Timezone      types.String `tfsdk:"timezone"`
	Provisioned   types.Bool   `tfsdk:"provisioned"`
	GUIAccess     types.String `tfsdk:"gui_access"`
	DebugMode     types.String `tfsdk:"debug_mode"`
	UsersStatus   types.String `tfsdk:"users_status"`
	RoleID        types.String `tfsdk:"role_id"`
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
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the page to redirect to after logging in.",
			},
			"auto_login": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether auto-login is enabled for the user.",
			},
			"auto_logout": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Idle time before auto-logout. Accepts seconds and time suffix (e.g. `30s`). `0` disables auto-logout.",
			},
			"language": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Language code for the user's interface, or `default` to use the system language.",
			},
			"refresh": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Automatic page refresh interval. Accepts seconds and time suffix (e.g. `30s`).",
			},
			"theme": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User interface theme: `default`, `blue-theme`, `dark-theme`, etc.",
			},
			"attempt_failed": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Number of consecutive failed login attempts.",
			},
			"attempt_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IP address from which the last failed login was attempted.",
			},
			"attempt_clock": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp of the last failed login attempt.",
			},
			"timezone": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's timezone, or `default` to use the system timezone.",
			},
			"provisioned": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the user was provisioned by an external directory.",
			},
			"gui_access": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Frontend authentication method inherited from user groups: `system_default`, `internal`, or `disabled`.",
			},
			"debug_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Debug mode status inherited from user groups: `disabled` or `enabled`.",
			},
			"users_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User account status: `enabled` or `disabled`.",
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

func populateUserModel(data *UserDataSourceModel, user *client.User) {
	data.ID = types.StringValue(user.UserID)
	data.Username = types.StringValue(user.Username)
	data.Name = types.StringValue(user.Name)
	data.Surname = types.StringValue(user.Surname)
	data.URL = types.StringValue(user.URL)
	data.AutoLogin = types.BoolValue(user.AutoLogin == "1")
	data.AutoLogout = types.StringValue(user.AutoLogout)
	data.Language = types.StringValue(user.Language)
	data.Refresh = types.StringValue(user.Refresh)
	data.Theme = types.StringValue(user.Theme)
	data.AttemptFailed = types.StringValue(user.AttemptFailed)
	data.AttemptIP = types.StringValue(user.AttemptIP)
	data.AttemptClock = types.StringValue(user.AttemptClock)
	data.Timezone = types.StringValue(user.Timezone)
	data.Provisioned = types.BoolValue(user.Provisioned == "1")
	data.GUIAccess = types.StringValue(guiAccessReverseMap[user.GUIAccess])
	data.DebugMode = types.StringValue(debugModeReverseMap[user.DebugMode])
	data.UsersStatus = types.StringValue(usersStatusReverseMap[user.UsersStatus])
	data.RoleID = types.StringValue(user.RoleID)
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
		populateUserModel(&data, user)
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
			populateUserModel(&data, &users[0])
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
