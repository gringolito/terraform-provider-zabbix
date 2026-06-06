package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &TemplateResource{}
var _ resource.ResourceWithImportState = &TemplateResource{}

func NewTemplateResource() resource.Resource {
	return &TemplateResource{}
}

type TemplateResource struct {
	client client.Client
}

type TemplateResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Host              types.String `tfsdk:"host"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	TemplateGroupIDs  types.Set    `tfsdk:"template_group_ids"`
	Macros            types.Map    `tfsdk:"macros"`
	LinkedTemplateIDs types.Set    `tfsdk:"linked_template_ids"`
}

func (r *TemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (r *TemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyMap, _ := types.MapValue(types.StringType, map[string]attr.Value{})
	emptySet, _ := types.SetValue(types.StringType, []attr.Value{})

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the template.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Technical name of the template. Must be unique within Zabbix.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Visible display name of the template. Defaults to the technical name if not set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Description of the template.",
			},
			"template_group_ids": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of template group IDs the template belongs to. At least one is required. This set is authoritative — any groups not listed here are removed on apply.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"macros": schema.MapAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             mapdefault.StaticValue(emptyMap),
				MarkdownDescription: "Map of user macro names to values. Macro names must use the `{$NAME}` format. This map is authoritative — any macros not listed here are removed on apply.",
			},
			"linked_template_ids": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(emptySet),
				MarkdownDescription: "Set of template IDs this template links to (inherits from). This set is authoritative — any linked templates not listed here are unlinked on apply.",
			},
		},
	}
}

func (r *TemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := modelToClientTemplate(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.TemplateCreate(ctx, r.client, t)
	if err != nil {
		resp.Diagnostics.AddError("Error creating template", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.TemplateGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading template after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(clientTemplateToModel(ctx, *created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, err := client.TemplateGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading template", err.Error())
		return
	}
	if t == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(clientTemplateToModel(ctx, *t, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := modelToClientTemplate(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.TemplateUpdate(ctx, r.client, t); err != nil {
		resp.Diagnostics.AddError("Error updating template", err.Error())
		return
	}

	updated, err := client.TemplateGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading template after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(clientTemplateToModel(ctx, *updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.TemplateDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting template", err.Error())
		return
	}
}

func (r *TemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func modelToClientTemplate(ctx context.Context, data TemplateResourceModel) (client.Template, diag.Diagnostics) {
	var diags diag.Diagnostics

	var groupIDStrings []string
	diags.Append(data.TemplateGroupIDs.ElementsAs(ctx, &groupIDStrings, false)...)
	if diags.HasError() {
		return client.Template{}, diags
	}
	groups := make([]client.TemplateGroupRef, len(groupIDStrings))
	for i, id := range groupIDStrings {
		groups[i] = client.TemplateGroupRef{GroupID: id}
	}

	var macroMap map[string]string
	if !data.Macros.IsNull() && !data.Macros.IsUnknown() {
		diags.Append(data.Macros.ElementsAs(ctx, &macroMap, false)...)
		if diags.HasError() {
			return client.Template{}, diags
		}
	}
	macros := make([]client.TemplateMacro, 0, len(macroMap))
	for macro, value := range macroMap {
		macros = append(macros, client.TemplateMacro{Macro: macro, Value: value})
	}

	var linkedIDStrings []string
	if !data.LinkedTemplateIDs.IsNull() && !data.LinkedTemplateIDs.IsUnknown() {
		diags.Append(data.LinkedTemplateIDs.ElementsAs(ctx, &linkedIDStrings, false)...)
		if diags.HasError() {
			return client.Template{}, diags
		}
	}
	parentTemplates := make([]client.TemplateRef, len(linkedIDStrings))
	for i, id := range linkedIDStrings {
		parentTemplates[i] = client.TemplateRef{TemplateID: id}
	}

	name := data.Name.ValueString()
	if name == "" {
		name = data.Host.ValueString()
	}

	return client.Template{
		TemplateID:      data.ID.ValueString(),
		Host:            data.Host.ValueString(),
		Name:            name,
		Description:     data.Description.ValueString(),
		Groups:          groups,
		Macros:          macros,
		ParentTemplates: parentTemplates,
	}, diags
}

func clientTemplateToModel(_ context.Context, t client.Template, data *TemplateResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Host = types.StringValue(t.Host)
	data.Name = types.StringValue(t.Name)
	data.Description = types.StringValue(t.Description)

	groupIDVals := make([]attr.Value, len(t.Groups))
	for i, g := range t.Groups {
		groupIDVals[i] = types.StringValue(g.GroupID)
	}
	groupSet, d := types.SetValue(types.StringType, groupIDVals)
	diags.Append(d...)
	if !d.HasError() {
		data.TemplateGroupIDs = groupSet
	}

	macroVals := make(map[string]attr.Value, len(t.Macros))
	for _, m := range t.Macros {
		macroVals[m.Macro] = types.StringValue(m.Value)
	}
	macroMap, d := types.MapValue(types.StringType, macroVals)
	diags.Append(d...)
	if !d.HasError() {
		data.Macros = macroMap
	}

	linkedIDVals := make([]attr.Value, len(t.ParentTemplates))
	for i, ref := range t.ParentTemplates {
		linkedIDVals[i] = types.StringValue(ref.TemplateID)
	}
	linkedSet, d := types.SetValue(types.StringType, linkedIDVals)
	diags.Append(d...)
	if !d.HasError() {
		data.LinkedTemplateIDs = linkedSet
	}

	return diags
}
