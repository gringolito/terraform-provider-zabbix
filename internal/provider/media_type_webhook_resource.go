package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MediaTypeWebhookResource{}
var _ resource.ResourceWithImportState = &MediaTypeWebhookResource{}

func NewMediaTypeWebhookResource() resource.Resource {
	return &MediaTypeWebhookResource{}
}

type MediaTypeWebhookResource struct {
	client client.Client
}

type MediaTypeWebhookModel struct {
	MediaTypeBaseModel
	Script        types.String `tfsdk:"script"`
	Timeout       types.String `tfsdk:"timeout"`
	ProcessTags   types.Bool   `tfsdk:"process_tags"`
	ShowEventMenu types.Bool   `tfsdk:"show_event_menu"`
	EventMenuURL  types.String `tfsdk:"event_menu_url"`
	EventMenuName types.String `tfsdk:"event_menu_name"`
	// types.List handles null/unknown during ImportState; element type: WebhookParameterModel
	Parameters types.List `tfsdk:"parameters"`
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

func (r *MediaTypeWebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_webhook"
}

func (r *MediaTypeWebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonMediaTypeResourceAttributes()
	attrs["script"] = rschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "JavaScript body of the webhook.",
	}
	attrs["timeout"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("30s"),
		MarkdownDescription: "Script execution timeout (e.g. `30s`). Defaults to `30s`.",
	}
	attrs["process_tags"] = rschema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "Whether to add event tags from webhook response. Defaults to `false`.",
	}
	attrs["show_event_menu"] = rschema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "Whether to add a link to the event menu. Defaults to `false`.",
	}
	attrs["event_menu_url"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "URL for the event menu entry. Used when `show_event_menu` is `true`.",
	}
	attrs["event_menu_name"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "Label for the event menu entry. Used when `show_event_menu` is `true`.",
	}
	attrs["parameters"] = rschema.ListNestedAttribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "Key/value pairs passed to the webhook script.",
		NestedObject: rschema.NestedAttributeObject{
			Attributes: map[string]rschema.Attribute{
				"name": rschema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Parameter name.",
				},
				"value": rschema.StringAttribute{
					Required:            true,
					Sensitive:           true,
					MarkdownDescription: "Parameter value. Sensitive.",
				},
			},
		},
	}
	resp.Schema = rschema.Schema{
		MarkdownDescription: "Manages a Zabbix webhook media type (notification channel via JavaScript).",
		Attributes:          attrs,
	}
}

func (r *MediaTypeWebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *MediaTypeWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MediaTypeWebhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save parameter values before read-after-write clears them (API does not echo values).
	savedParams := data.Parameters

	mt, diags := webhookModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.MediaTypeCreate(ctx, r.client, mt)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook media type", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.MediaTypeGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook media type after create", err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("Webhook media type not found after create",
			"Zabbix returned no media type immediately after creation.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToWebhookModel(ctx, created, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(restoreWebhookParamValues(ctx, &data, savedParams)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MediaTypeWebhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook media type", err.Error())
		return
	}
	if mt == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	savedParams := data.Parameters
	resp.Diagnostics.Append(mediaTypeToWebhookModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(restoreWebhookParamValues(ctx, &data, savedParams)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MediaTypeWebhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MediaTypeWebhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge state values into plan for any parameter whose value wasn't changed.
	resp.Diagnostics.Append(restoreWebhookParamValues(ctx, &data, state.Parameters)...)
	if resp.Diagnostics.HasError() {
		return
	}
	savedParams := data.Parameters

	mt, diags := webhookModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.MediaTypeUpdate(ctx, r.client, mt); err != nil {
		resp.Diagnostics.AddError("Error updating webhook media type", err.Error())
		return
	}

	updated, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook media type after update", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("Webhook media type not found after update",
			"Zabbix returned no media type immediately after update.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToWebhookModel(ctx, updated, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(restoreWebhookParamValues(ctx, &data, savedParams)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MediaTypeWebhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.MediaTypeDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting webhook media type", err.Error())
	}
}

func (r *MediaTypeWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Conversion helpers ---

func webhookModelToMediaType(ctx context.Context, m MediaTypeWebhookModel) (client.MediaType, diag.Diagnostics) {
	mt, diags := mediaTypeBaseFromModel(ctx, m.MediaTypeBaseModel)
	mt.Type = client.MediaTypeTypeWebhook
	mt.Script = m.Script.ValueString()
	mt.Timeout = m.Timeout.ValueString()
	mt.ProcessTags = boolToInt(m.ProcessTags.ValueBool())
	mt.ShowEventMenu = boolToInt(m.ShowEventMenu.ValueBool())
	mt.EventMenuURL = m.EventMenuURL.ValueString()
	mt.EventMenuName = m.EventMenuName.ValueString()
	if !m.Parameters.IsNull() && !m.Parameters.IsUnknown() {
		var paramModels []WebhookParameterModel
		diags.Append(m.Parameters.ElementsAs(ctx, &paramModels, false)...)
		if !diags.HasError() {
			params := make([]client.MediaTypeParameter, len(paramModels))
			for i, p := range paramModels {
				params[i] = client.MediaTypeParameter{
					Name:  p.Name.ValueString(),
					Value: p.Value.ValueString(),
				}
			}
			mt.Parameters = params
		}
	}
	return mt, diags
}

// mediaTypeToWebhookModel populates the webhook model from an API response.
// Parameter values are set to "" — callers must restore them from prior state or plan
// since the API does not return sensitive values.
func mediaTypeToWebhookModel(ctx context.Context, mt *client.MediaType, m *MediaTypeWebhookModel) diag.Diagnostics {
	diags := mediaTypeBaseToModel(ctx, mt, &m.MediaTypeBaseModel)
	m.Script = types.StringValue(mt.Script)
	m.Timeout = types.StringValue(mt.Timeout)
	m.ProcessTags = types.BoolValue(intToBool(mt.ProcessTags))
	m.ShowEventMenu = types.BoolValue(intToBool(mt.ShowEventMenu))
	m.EventMenuURL = types.StringValue(mt.EventMenuURL)
	m.EventMenuName = types.StringValue(mt.EventMenuName)

	paramModels := make([]WebhookParameterModel, len(mt.Parameters))
	for i, p := range mt.Parameters {
		paramModels[i] = WebhookParameterModel{
			Name:  types.StringValue(p.Name),
			Value: types.StringValue(""), // sensitive — not echoed by API
		}
	}
	var d diag.Diagnostics
	m.Parameters, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: webhookParamAttrTypes}, paramModels)
	diags.Append(d...)
	return diags
}

// restoreWebhookParamValues copies sensitive parameter values from prior into model,
// matching by parameter name. Parameters not in prior keep their current value.
func restoreWebhookParamValues(ctx context.Context, m *MediaTypeWebhookModel, prior types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	if prior.IsNull() || prior.IsUnknown() || m.Parameters.IsNull() || m.Parameters.IsUnknown() {
		return diags
	}

	var currentParams, priorParams []WebhookParameterModel
	diags.Append(m.Parameters.ElementsAs(ctx, &currentParams, false)...)
	diags.Append(prior.ElementsAs(ctx, &priorParams, false)...)
	if diags.HasError() {
		return diags
	}

	priorValues := make(map[string]types.String, len(priorParams))
	for _, p := range priorParams {
		priorValues[p.Name.ValueString()] = p.Value
	}
	for i, p := range currentParams {
		if v, ok := priorValues[p.Name.ValueString()]; ok {
			currentParams[i].Value = v
		}
	}

	var d diag.Diagnostics
	m.Parameters, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: webhookParamAttrTypes}, currentParams)
	diags.Append(d...)
	return diags
}
