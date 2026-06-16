package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ActionTriggerDataSource{}
var _ datasource.DataSourceWithConfigure = &ActionTriggerDataSource{}

func NewActionTriggerDataSource() datasource.DataSource {
	return &ActionTriggerDataSource{}
}

type ActionTriggerDataSource struct {
	client client.Client
}

type ActionTriggerDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

func (d *ActionTriggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_trigger"
}

func (d *ActionTriggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Zabbix trigger action by `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the trigger action. Exactly one of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the trigger action. Exactly one of `id` or `name` must be set.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Status of the trigger action. One of: `enabled`, `disabled`.",
			},
		},
	}
}

func (d *ActionTriggerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ActionTriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActionTriggerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != ""
	nameSet := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""

	if !idSet && !nameSet {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Exactly one of \"id\" or \"name\" must be set.",
		)
		return
	}

	var actions []client.Action

	if idSet {
		a, err := client.ActionGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger action", err.Error())
			return
		}
		if a == nil {
			resp.Diagnostics.AddError(
				"Trigger action not found",
				fmt.Sprintf("No trigger action found with id %q.", data.ID.ValueString()),
			)
			return
		}
		actions = []client.Action{*a}
	} else {
		var err error
		actions, err = client.ActionGetByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger action", err.Error())
			return
		}
		if len(actions) == 0 {
			resp.Diagnostics.AddError(
				"Trigger action not found",
				fmt.Sprintf("No trigger action found with name %q.", data.Name.ValueString()),
			)
			return
		}
		if len(actions) > 1 {
			resp.Diagnostics.AddError(
				"Multiple trigger actions found",
				fmt.Sprintf("Found %d trigger actions with name %q. Use \"id\" to disambiguate.", len(actions), data.Name.ValueString()),
			)
			return
		}
	}

	a := actions[0]
	data.ID = types.StringValue(a.ActionID)
	data.Name = types.StringValue(a.Name)

	status, ok := actionStatusReverseMap[a.Status]
	if !ok {
		resp.Diagnostics.AddError("Unknown action status", fmt.Sprintf("Unrecognized status %d from API.", a.Status))
		return
	}
	data.Status = types.StringValue(status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
