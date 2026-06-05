package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HostDataSource{}

func NewHostDataSource() datasource.DataSource {
	return &HostDataSource{}
}

type HostDataSource struct {
	client client.Client
}

type HostDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Host          types.String `tfsdk:"host"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Status        types.String `tfsdk:"status"`
	HostGroupIDs  types.Set    `tfsdk:"host_group_ids"`
	Tags          types.Set    `tfsdk:"tags"`
	InventoryMode types.String `tfsdk:"inventory_mode"`
	ProxyID       types.String `tfsdk:"proxy_id"`
}

func (d *HostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (d *HostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix host by ID or technical name. Exactly one of `id` or `host` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the host. One of `id` or `host` must be set.",
			},
			"host": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Technical name of the host. One of `id` or `host` must be set.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Visible display name of the host.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the host.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Monitoring status of the host. One of: `enabled`, `disabled`.",
			},
			"host_group_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of host group IDs the host belongs to.",
			},
			"tags": schema.SetNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Tags attached to the host.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Tag name.",
						},
						"value": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Tag value.",
						},
					},
				},
			},
			"inventory_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Inventory population mode. One of: `disabled`, `manual`, `automatic`.",
			},
			"proxy_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the proxy monitoring the host, or `0` for no proxy.",
			},
		},
	}
}

func (d *HostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HostDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Host.IsNull() {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Exactly one of `id` or `host` must be set.",
		)
		return
	}

	var h *client.Host

	if !data.ID.IsNull() {
		found, err := client.HostGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading host", err.Error())
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Host not found",
				fmt.Sprintf("No host found with id %q.", data.ID.ValueString()),
			)
			return
		}
		h = found
	} else {
		hosts, err := client.HostGetByTechnicalName(ctx, d.client, data.Host.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading host", err.Error())
			return
		}
		switch len(hosts) {
		case 0:
			resp.Diagnostics.AddError(
				"Host not found",
				fmt.Sprintf("No host found with technical name %q.", data.Host.ValueString()),
			)
			return
		case 1:
			h = &hosts[0]
		default:
			resp.Diagnostics.AddError(
				"Multiple hosts found",
				fmt.Sprintf("Found %d hosts with technical name %q; use `id` to disambiguate.", len(hosts), data.Host.ValueString()),
			)
			return
		}
	}

	data.ID = types.StringValue(h.HostID)
	dsModel := &HostResourceModel{
		ID:           data.ID,
		HostGroupIDs: data.HostGroupIDs,
		Tags:         data.Tags,
	}
	resp.Diagnostics.Append(clientHostToModel(ctx, *h, dsModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Host = dsModel.Host
	data.Name = dsModel.Name
	data.Description = dsModel.Description
	data.Status = dsModel.Status
	data.HostGroupIDs = dsModel.HostGroupIDs
	data.Tags = dsModel.Tags
	data.InventoryMode = dsModel.InventoryMode
	data.ProxyID = dsModel.ProxyID

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
