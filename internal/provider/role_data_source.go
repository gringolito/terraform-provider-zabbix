package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RoleDataSource{}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

type RoleDataSource struct {
	client client.Client
}

type RoleDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Type  types.String `tfsdk:"type"`
	Rules types.Object `tfsdk:"rules"`
}

func (d *RoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix role by ID or name. Exactly one of `id` or `name` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the role. One of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display name of the role. One of `id` or `name` must be set.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Role type: `user`, `admin`, or `super_admin`.",
			},
			"rules": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access rules for the role.",
				Attributes: map[string]schema.Attribute{
					"ui": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "UI element access rules.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Name of the UI element.",
								},
								"enabled": schema.BoolAttribute{
									Computed:            true,
									MarkdownDescription: "Whether access to this UI element is enabled.",
								},
							},
						},
					},
					"ui_default_access": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Default access for UI elements not listed in `ui`.",
					},
					"modules_default_access": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Default access for modules not listed in `modules`.",
					},
					"actions_default_access": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Default access for actions not listed in `actions`.",
					},
					"api_access": schema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether API access is enabled for this role.",
					},
					"api_mode": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "API access mode: `deny` or `allow`.",
					},
					"api_methods": schema.SetAttribute{
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "List of API methods affected by `api_mode`.",
					},
					"modules": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Module access rules.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"module_id": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Module ID.",
								},
								"enabled": schema.BoolAttribute{
									Computed:            true,
									MarkdownDescription: "Whether access to this module is enabled.",
								},
							},
						},
					},
					"actions": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Action access rules.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Name of the action.",
								},
								"enabled": schema.BoolAttribute{
									Computed:            true,
									MarkdownDescription: "Whether this action is enabled.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *RoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel
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
		role, err := client.RoleGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading role", err.Error())
			return
		}
		if role == nil {
			resp.Diagnostics.AddError(
				"Role not found",
				fmt.Sprintf("No role found with id %q.", data.ID.ValueString()),
			)
			return
		}
		data.ID = types.StringValue(role.ID)
		data.Name = types.StringValue(role.Name)
		data.Type = types.StringValue(roleTypeReverseMap[role.Type])
		rulesObj, diags := clientRulesToObject(role.Rules)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Rules = rulesObj
	} else {
		roles, err := client.RoleGetByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading role", err.Error())
			return
		}
		switch len(roles) {
		case 0:
			resp.Diagnostics.AddError(
				"Role not found",
				fmt.Sprintf("No role found with name %q.", data.Name.ValueString()),
			)
			return
		case 1:
			data.ID = types.StringValue(roles[0].ID)
			data.Name = types.StringValue(roles[0].Name)
			data.Type = types.StringValue(roleTypeReverseMap[roles[0].Type])
			rulesObj, diags := clientRulesToObject(roles[0].Rules)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.Rules = rulesObj
		default:
			resp.Diagnostics.AddError(
				"Multiple roles found",
				fmt.Sprintf("Found %d roles with name %q; use `id` to disambiguate.", len(roles), data.Name.ValueString()),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
