package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ItemDataSource{}

func NewItemDataSource() datasource.DataSource {
	return &ItemDataSource{}
}

type ItemDataSource struct {
	client client.Client
}

type ItemDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Key        types.String `tfsdk:"key_"`
	Name       types.String `tfsdk:"name"`
	HostID     types.String `tfsdk:"host_id"`
	TemplateID types.String `tfsdk:"template_id"`
}

func (d *ItemDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item"
}

func (d *ItemDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix item by `id`, or by `key_` + exactly one of `host_id`/`template_id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the item. One of `id` or (`key_` + scope) must be set.",
			},
			"key_": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Item key. Used with `host_id` or `template_id` for scoped lookup.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Display name of the item.",
			},
			"host_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the host to scope the lookup. Used with `key_`.",
			},
			"template_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the template to scope the lookup. Used with `key_`.",
			},
		},
	}
}

func (d *ItemDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ItemDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ItemDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull()
	hasKey := !data.Key.IsNull()
	hasHostID := !data.HostID.IsNull()
	hasTemplateID := !data.TemplateID.IsNull()

	if !hasID && !hasKey {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Provide either `id`, or `key_` with exactly one of `host_id` or `template_id`.",
		)
		return
	}

	if hasKey && !hasHostID && !hasTemplateID {
		resp.Diagnostics.AddError(
			"Missing scope",
			"When using `key_` lookup, provide exactly one of `host_id` or `template_id`.",
		)
		return
	}

	if hasKey && hasHostID && hasTemplateID {
		resp.Diagnostics.AddError(
			"Ambiguous scope",
			"Provide exactly one of `host_id` or `template_id`, not both.",
		)
		return
	}

	var item *client.Item

	if hasID {
		found, err := client.ItemGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading item", err.Error())
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Item not found",
				fmt.Sprintf("No item found with id %q.", data.ID.ValueString()),
			)
			return
		}
		item = found
	} else {
		hostID := ""
		templateID := ""
		if hasHostID {
			hostID = data.HostID.ValueString()
		} else {
			templateID = data.TemplateID.ValueString()
		}

		items, err := client.ItemGetByKeyAndScope(ctx, d.client, data.Key.ValueString(), hostID, templateID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading item", err.Error())
			return
		}

		switch len(items) {
		case 0:
			if hasHostID {
				resp.Diagnostics.AddError(
					"Item not found",
					fmt.Sprintf("Found 0 items with key_ %q on host id %q.", data.Key.ValueString(), hostID),
				)
			} else {
				resp.Diagnostics.AddError(
					"Item not found",
					fmt.Sprintf("Found 0 items with key_ %q on template id %q.", data.Key.ValueString(), templateID),
				)
			}
			return
		case 1:
			item = &items[0]
		default:
			if hasHostID {
				resp.Diagnostics.AddError(
					"Multiple items found",
					fmt.Sprintf("Found %d items with key_ %q on host id %q; use `id` to disambiguate.", len(items), data.Key.ValueString(), hostID),
				)
			} else {
				resp.Diagnostics.AddError(
					"Multiple items found",
					fmt.Sprintf("Found %d items with key_ %q on template id %q; use `id` to disambiguate.", len(items), data.Key.ValueString(), templateID),
				)
			}
			return
		}
	}

	data.ID = types.StringValue(item.ItemID)
	data.Key = types.StringValue(item.Key)
	data.Name = types.StringValue(item.Name)
	data.HostID = types.StringValue(item.HostID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
