package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TemplateDataSource{}

func NewTemplateDataSource() datasource.DataSource {
	return &TemplateDataSource{}
}

type TemplateDataSource struct {
	client client.Client
}

type TemplateDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Host              types.String `tfsdk:"host"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	TemplateGroupIDs  types.Set    `tfsdk:"template_group_ids"`
	Macros            types.Map    `tfsdk:"macros"`
	LinkedTemplateIDs types.Set    `tfsdk:"linked_template_ids"`
}

func (d *TemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *TemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix template by ID or technical name. Exactly one of `id` or `host` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the template. One of `id` or `host` must be set.",
			},
			"host": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Technical name of the template. One of `id` or `host` must be set.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Visible display name of the template.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the template.",
			},
			"template_group_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of template group IDs the template belongs to.",
			},
			"macros": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Map of user macro names to values.",
			},
			"linked_template_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of template IDs this template links to (inherits from).",
			},
		},
	}
}

func (d *TemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TemplateDataSourceModel
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

	var t *client.Template

	if !data.ID.IsNull() {
		found, err := client.TemplateGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading template", err.Error())
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Template not found",
				fmt.Sprintf("No template found with id %q.", data.ID.ValueString()),
			)
			return
		}
		t = found
	} else {
		templates, err := client.TemplateGetByHost(ctx, d.client, data.Host.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading template", err.Error())
			return
		}
		switch len(templates) {
		case 0:
			resp.Diagnostics.AddError(
				"Template not found",
				fmt.Sprintf("No template found with technical name %q.", data.Host.ValueString()),
			)
			return
		case 1:
			t = &templates[0]
		default:
			resp.Diagnostics.AddError(
				"Multiple templates found",
				fmt.Sprintf("Found %d templates with technical name %q; use `id` to disambiguate.", len(templates), data.Host.ValueString()),
			)
			return
		}
	}

	data.ID = types.StringValue(t.TemplateID)
	rm := &TemplateResourceModel{
		ID:               data.ID,
		TemplateGroupIDs: data.TemplateGroupIDs,
		Macros:           data.Macros,
	}
	resp.Diagnostics.Append(clientTemplateToModel(ctx, *t, rm)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Host = rm.Host
	data.Name = rm.Name
	data.Description = rm.Description
	data.TemplateGroupIDs = rm.TemplateGroupIDs
	data.Macros = rm.Macros

	linkedIDVals := make([]attr.Value, len(t.ParentTemplates))
	for i, ref := range t.ParentTemplates {
		linkedIDVals[i] = types.StringValue(ref.TemplateID)
	}
	linkedSet, dSet := types.SetValue(types.StringType, linkedIDVals)
	resp.Diagnostics.Append(dSet...)
	if !dSet.HasError() {
		data.LinkedTemplateIDs = linkedSet
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
