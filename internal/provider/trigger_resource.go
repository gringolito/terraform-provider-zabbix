package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &TriggerResource{}
var _ resource.ResourceWithImportState = &TriggerResource{}
var _ resource.ResourceWithConfigValidators = &TriggerResource{}

var (
	triggerRecoveryModeMap = map[string]int64{
		"expression": 0, "recovery_expression": 1, "none": 2,
	}
	triggerRecoveryModeReverseMap = map[int64]string{
		0: "expression", 1: "recovery_expression", 2: "none",
	}
	triggerPriorityMap = map[string]int64{
		"not_classified": 0, "information": 1, "warning": 2,
		"average": 3, "high": 4, "disaster": 5,
	}
	triggerPriorityReverseMap = map[int64]string{
		0: "not_classified", 1: "information", 2: "warning",
		3: "average", 4: "high", 5: "disaster",
	}
	triggerStatusMap = map[string]int64{
		"enabled": 0, "disabled": 1,
	}
	triggerStatusReverseMap = map[int64]string{
		0: "enabled", 1: "disabled",
	}
)

func NewTriggerResource() resource.Resource {
	return &TriggerResource{}
}

type TriggerResource struct {
	client client.Client
}

type TriggerResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Description        types.String `tfsdk:"description"`
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

func (r *TriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

func (r *TriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix trigger.\n\n" +
			"~> **Note:** Unlike other child resources, `zabbix_trigger` does not carry a `host_id` or `template_id`. " +
			"Ownership is inferred by Zabbix from the expression itself (e.g. `last(/web01/cpu.util)>90`). " +
			"Wire Terraform dependency edges via expression interpolation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the trigger.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the trigger.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"expression": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Trigger expression. Must reference at least one item using Zabbix expression syntax.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"recovery_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("expression"),
				MarkdownDescription: "Recovery mode. One of: `expression`, `recovery_expression`, `none`. Defaults to `expression`.",
				Validators: []validator.String{
					stringvalidator.OneOf("expression", "recovery_expression", "none"),
				},
			},
			"recovery_expression": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Recovery expression. Required when `recovery_mode = \"recovery_expression\"`, must be empty otherwise.",
			},
			"priority": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Severity of the trigger. One of: `not_classified`, `information`, `warning`, `average`, `high`, `disaster`.",
				Validators: []validator.String{
					stringvalidator.OneOf("not_classified", "information", "warning", "average", "high", "disaster"),
				},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Monitoring status of the trigger. One of: `enabled`, `disabled`. Defaults to `enabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"manual_close": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the problem can be manually closed. Defaults to `false`.",
			},
			"comments": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Additional description/comments for the trigger.",
			},
			"url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "URL associated with the trigger. Zabbix accepts relative paths and macro-based URLs.",
			},
			"tags": schema.SetNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Set of tags to attach to the trigger.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Tag name.",
						},
						"value": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(""),
							MarkdownDescription: "Tag value. Defaults to `\"\"`.",
						},
					},
				},
			},
		},
	}
}

func (r *TriggerResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		triggerRecoveryExpressionValidator{},
	}
}

// triggerRecoveryExpressionValidator enforces:
//   - recovery_expression must be non-empty when recovery_mode = "recovery_expression"
//   - recovery_expression must be empty when recovery_mode != "recovery_expression"
type triggerRecoveryExpressionValidator struct{}

func (v triggerRecoveryExpressionValidator) Description(_ context.Context) string {
	return `"recovery_expression" must be set iff "recovery_mode" = "recovery_expression".`
}

func (v triggerRecoveryExpressionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v triggerRecoveryExpressionValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data TriggerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.RecoveryMode.IsUnknown() || data.RecoveryExpression.IsUnknown() {
		return
	}

	mode := data.RecoveryMode.ValueString()
	expr := data.RecoveryExpression.ValueString()

	if mode == "recovery_expression" && expr == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("recovery_expression"),
			"Missing recovery_expression",
			`"recovery_expression" must be set when "recovery_mode" = "recovery_expression".`,
		)
	}
	if mode != "recovery_expression" && expr != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("recovery_expression"),
			"Unexpected recovery_expression",
			fmt.Sprintf(`"recovery_expression" must be empty when "recovery_mode" = %q.`, mode),
		)
	}
}

func (r *TriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tr, diags := modelToClientTrigger(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.TriggerCreate(ctx, r.client, tr)
	if err != nil {
		resp.Diagnostics.AddError("Error creating trigger", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.TriggerGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(clientTriggerToModel(ctx, *created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tr, err := client.TriggerGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger", err.Error())
		return
	}
	if tr == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(clientTriggerToModel(ctx, *tr, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	tr, diags := modelToClientTrigger(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.TriggerUpdate(ctx, r.client, tr); err != nil {
		resp.Diagnostics.AddError("Error updating trigger", err.Error())
		return
	}

	updated, err := client.TriggerGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(clientTriggerToModel(ctx, *updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.TriggerDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting trigger", err.Error())
		return
	}
}

func (r *TriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelToClientTrigger converts a TriggerResourceModel to a client.Trigger.
func modelToClientTrigger(ctx context.Context, data TriggerResourceModel) (client.Trigger, diag.Diagnostics) {
	var diags diag.Diagnostics

	manualClose := int64(0)
	if data.ManualClose.ValueBool() {
		manualClose = 1
	}

	tr := client.Trigger{
		TriggerID:          data.ID.ValueString(),
		Description:        data.Description.ValueString(),
		Expression:         data.Expression.ValueString(),
		RecoveryMode:       triggerRecoveryModeMap[data.RecoveryMode.ValueString()],
		RecoveryExpression: data.RecoveryExpression.ValueString(),
		Priority:           triggerPriorityMap[data.Priority.ValueString()],
		Status:             triggerStatusMap[data.Status.ValueString()],
		ManualClose:        manualClose,
		Comments:           data.Comments.ValueString(),
		URL:                data.URL.ValueString(),
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tagModels []TagModel
		diags.Append(data.Tags.ElementsAs(ctx, &tagModels, false)...)
		if diags.HasError() {
			return tr, diags
		}
		tr.Tags = make([]client.TriggerTag, len(tagModels))
		for i, tm := range tagModels {
			tr.Tags[i] = client.TriggerTag{
				Tag:   tm.Name.ValueString(),
				Value: tm.Value.ValueString(),
			}
		}
	} else {
		tr.Tags = []client.TriggerTag{}
	}

	return tr, diags
}

// clientTriggerToModel updates a TriggerResourceModel from a client.Trigger.
func clientTriggerToModel(_ context.Context, tr client.Trigger, data *TriggerResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Description = types.StringValue(tr.Description)
	data.Expression = types.StringValue(tr.Expression)
	data.RecoveryExpression = types.StringValue(tr.RecoveryExpression)
	data.Comments = types.StringValue(tr.Comments)
	data.URL = types.StringValue(tr.URL)
	data.ManualClose = types.BoolValue(tr.ManualClose == 1)

	recoveryMode, ok := triggerRecoveryModeReverseMap[tr.RecoveryMode]
	if !ok {
		diags.AddError("Unknown recovery mode", fmt.Sprintf("Unrecognized recovery_mode %d from API.", tr.RecoveryMode))
		return diags
	}
	data.RecoveryMode = types.StringValue(recoveryMode)

	priority, ok := triggerPriorityReverseMap[tr.Priority]
	if !ok {
		diags.AddError("Unknown priority", fmt.Sprintf("Unrecognized priority %d from API.", tr.Priority))
		return diags
	}
	data.Priority = types.StringValue(priority)

	status, ok := triggerStatusReverseMap[tr.Status]
	if !ok {
		diags.AddError("Unknown trigger status", fmt.Sprintf("Unrecognized status %d from API.", tr.Status))
		return diags
	}
	data.Status = types.StringValue(status)

	tagVals := make([]attr.Value, len(tr.Tags))
	for i, tag := range tr.Tags {
		obj, d := types.ObjectValue(tagAttrTypes, map[string]attr.Value{
			"name":  types.StringValue(tag.Tag),
			"value": types.StringValue(tag.Value),
		})
		diags.Append(d...)
		tagVals[i] = obj
	}
	tagsSet, d := types.SetValue(types.ObjectType{AttrTypes: tagAttrTypes}, tagVals)
	diags.Append(d...)
	if !d.HasError() {
		data.Tags = tagsSet
	}

	return diags
}
