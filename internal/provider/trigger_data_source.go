package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TriggerDataSource{}

func NewTriggerDataSource() datasource.DataSource {
	return &TriggerDataSource{}
}

type TriggerDataSource struct {
	client client.Client
}

type TriggerDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Description        types.String `tfsdk:"description"`
	HostID             types.String `tfsdk:"host_id"`
	TemplateID         types.String `tfsdk:"template_id"`
	Expression         types.String `tfsdk:"expression"`
	RecoveryMode       types.String `tfsdk:"recovery_mode"`
	RecoveryExpression types.String `tfsdk:"recovery_expression"`
	Priority           types.String `tfsdk:"priority"`
	Status             types.String `tfsdk:"status"`
	ManualClose        types.Bool   `tfsdk:"manual_close"`
	Comments           types.String `tfsdk:"comments"`
	URL                types.String `tfsdk:"url"`
	Tags               types.Set    `tfsdk:"tags"`
}

func (d *TriggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

func (d *TriggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix trigger by `id`, or by `description` + exactly one of `host_id`/`template_id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the trigger. One of `id` or (`description` + scope) must be set.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the trigger. Used with `host_id` or `template_id` for scoped lookup.",
			},
			"host_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the host to scope the lookup. Used with `description`.",
			},
			"template_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the template to scope the lookup. Used with `description`.",
			},
			"expression": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Trigger expression.",
			},
			"recovery_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Recovery mode.",
			},
			"recovery_expression": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Recovery expression.",
			},
			"priority": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Severity of the trigger.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Monitoring status of the trigger.",
			},
			"manual_close": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the problem can be manually closed.",
			},
			"comments": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Additional description/comments.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL associated with the trigger.",
			},
			"tags": schema.SetNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Tags attached to the trigger.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":  schema.StringAttribute{Computed: true, MarkdownDescription: "Tag name."},
						"value": schema.StringAttribute{Computed: true, MarkdownDescription: "Tag value."},
					},
				},
			},
		},
	}
}

func (d *TriggerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TriggerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull()
	hasDescription := !data.Description.IsNull()
	hasHostID := !data.HostID.IsNull()
	hasTemplateID := !data.TemplateID.IsNull()

	if !hasID && !hasDescription {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Provide either `id`, or `description` with exactly one of `host_id` or `template_id`.",
		)
		return
	}

	if hasDescription && !hasHostID && !hasTemplateID {
		resp.Diagnostics.AddError(
			"Missing scope",
			"When using `description` lookup, provide exactly one of `host_id` or `template_id`.",
		)
		return
	}

	if hasDescription && hasHostID && hasTemplateID {
		resp.Diagnostics.AddError(
			"Ambiguous scope",
			"Provide exactly one of `host_id` or `template_id`, not both.",
		)
		return
	}

	var tr *client.Trigger

	if hasID {
		found, err := client.TriggerGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger", err.Error())
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Trigger not found",
				fmt.Sprintf("No trigger found with id %q.", data.ID.ValueString()),
			)
			return
		}
		tr = found
	} else {
		hostID := ""
		templateID := ""
		if hasHostID {
			hostID = data.HostID.ValueString()
		} else {
			templateID = data.TemplateID.ValueString()
		}

		triggers, err := client.TriggerGetByDescriptionAndScope(ctx, d.client, data.Description.ValueString(), hostID, templateID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger", err.Error())
			return
		}

		switch len(triggers) {
		case 0:
			if hasHostID {
				resp.Diagnostics.AddError(
					"Trigger not found",
					fmt.Sprintf("Found 0 triggers with description %q on host id %q.", data.Description.ValueString(), hostID),
				)
			} else {
				resp.Diagnostics.AddError(
					"Trigger not found",
					fmt.Sprintf("Found 0 triggers with description %q on template id %q.", data.Description.ValueString(), templateID),
				)
			}
			return
		case 1:
			tr = &triggers[0]
		default:
			if hasHostID {
				resp.Diagnostics.AddError(
					"Multiple triggers found",
					fmt.Sprintf("Found %d triggers with description %q on host id %q; use `id` to disambiguate.", len(triggers), data.Description.ValueString(), hostID),
				)
			} else {
				resp.Diagnostics.AddError(
					"Multiple triggers found",
					fmt.Sprintf("Found %d triggers with description %q on template id %q; use `id` to disambiguate.", len(triggers), data.Description.ValueString(), templateID),
				)
			}
			return
		}
	}

	data.ID = types.StringValue(tr.TriggerID)
	rm := &TriggerResourceModel{
		ID:   data.ID,
		Tags: data.Tags,
	}
	resp.Diagnostics.Append(clientTriggerToModel(ctx, *tr, rm)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Description = rm.Description
	data.Expression = rm.Expression
	data.RecoveryMode = rm.RecoveryMode
	data.RecoveryExpression = rm.RecoveryExpression
	data.Priority = rm.Priority
	data.Status = rm.Status
	data.ManualClose = rm.ManualClose
	data.Comments = rm.Comments
	data.URL = rm.URL
	data.Tags = rm.Tags

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
