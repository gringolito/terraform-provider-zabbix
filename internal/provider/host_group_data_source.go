package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HostGroupDataSource{}

func NewHostGroupDataSource() datasource.DataSource {
	return &HostGroupDataSource{}
}

type HostGroupDataSource struct {
	client client.Client
}

type HostGroupDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *HostGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_group"
}

func (d *HostGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix host group by ID or name. Exactly one of `id` or `name` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the host group. One of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display name of the host group. One of `id` or `name` must be set.",
			},
		},
	}
}

func (d *HostGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HostGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HostGroupDataSourceModel
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
		group, err := client.HostGroupGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading host group", err.Error())
			return
		}
		if group == nil {
			resp.Diagnostics.AddError(
				"Host group not found",
				fmt.Sprintf("No host group found with id %q.", data.ID.ValueString()),
			)
			return
		}
		data.ID = types.StringValue(group.ID)
		data.Name = types.StringValue(group.Name)
	} else {
		groups, err := client.HostGroupGetByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading host group", err.Error())
			return
		}
		switch len(groups) {
		case 0:
			resp.Diagnostics.AddError(
				"Host group not found",
				fmt.Sprintf("No host group found with name %q.", data.Name.ValueString()),
			)
			return
		case 1:
			data.ID = types.StringValue(groups[0].ID)
			data.Name = types.StringValue(groups[0].Name)
		default:
			resp.Diagnostics.AddError(
				"Multiple host groups found",
				fmt.Sprintf("Found %d host groups with name %q; use `id` to disambiguate.", len(groups), data.Name.ValueString()),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
