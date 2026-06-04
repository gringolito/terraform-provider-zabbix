package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &UserGroupResource{}
var _ resource.ResourceWithImportState = &UserGroupResource{}

var (
	guiAccessMap = map[string]int64{
		"system_default": 0, "internal": 1, "disabled": 2,
	}
	guiAccessReverseMap = map[int64]string{
		0: "system_default", 1: "internal", 2: "disabled",
	}
	debugModeMap = map[string]int64{
		"disabled": 0, "enabled": 1,
	}
	debugModeReverseMap = map[int64]string{
		0: "disabled", 1: "enabled",
	}
	usersStatusMap = map[string]int64{
		"enabled": 0, "disabled": 1,
	}
	usersStatusReverseMap = map[int64]string{
		0: "enabled", 1: "disabled",
	}
)

func NewUserGroupResource() resource.Resource {
	return &UserGroupResource{}
}

type UserGroupResource struct {
	client client.Client
}

type UserGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	GUIAccess   types.String `tfsdk:"gui_access"`
	DebugMode   types.String `tfsdk:"debug_mode"`
	UsersStatus types.String `tfsdk:"users_status"`
}

func (r *UserGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (r *UserGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix user group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the user group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the user group. Must be unique within Zabbix.",
			},
			"gui_access": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("system_default"),
				MarkdownDescription: "Frontend authentication method. One of: `system_default`, `internal`, `disabled`. Defaults to `system_default`.",
			},
			"debug_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Debug mode for the group. One of: `disabled`, `enabled`. Defaults to `disabled`.",
			},
			"users_status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Status of the users in this group. One of: `enabled`, `disabled`. Defaults to `enabled`.",
			},
		},
	}
}

func (r *UserGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ug := client.UserGroup{
		Name:        data.Name.ValueString(),
		GUIAccess:   guiAccessMap[data.GUIAccess.ValueString()],
		DebugMode:   debugModeMap[data.DebugMode.ValueString()],
		UsersStatus: usersStatusMap[data.UsersStatus.ValueString()],
	}
	id, err := client.UserGroupCreate(ctx, r.client, ug)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user group", err.Error())
		return
	}
	data.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ug, err := client.UserGroupGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user group", err.Error())
		return
	}
	if ug == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.Name = types.StringValue(ug.Name)
	data.GUIAccess = types.StringValue(guiAccessReverseMap[ug.GUIAccess])
	data.DebugMode = types.StringValue(debugModeReverseMap[ug.DebugMode])
	data.UsersStatus = types.StringValue(usersStatusReverseMap[ug.UsersStatus])
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ug := client.UserGroup{
		ID:          data.ID.ValueString(),
		Name:        data.Name.ValueString(),
		GUIAccess:   guiAccessMap[data.GUIAccess.ValueString()],
		DebugMode:   debugModeMap[data.DebugMode.ValueString()],
		UsersStatus: usersStatusMap[data.UsersStatus.ValueString()],
	}
	if err := client.UserGroupUpdate(ctx, r.client, ug); err != nil {
		resp.Diagnostics.AddError("Error updating user group", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.UserGroupDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting user group", err.Error())
		return
	}
}

func (r *UserGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
