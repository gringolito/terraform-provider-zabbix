package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MediaTypeScriptResource{}
var _ resource.ResourceWithImportState = &MediaTypeScriptResource{}

func NewMediaTypeScriptResource() resource.Resource {
	return &MediaTypeScriptResource{}
}

type MediaTypeScriptResource struct {
	client client.Client
}

type MediaTypeScriptModel struct {
	MediaTypeBaseModel
	ExecPath   types.String `tfsdk:"exec_path"`
	ExecParams types.String `tfsdk:"exec_params"`
}

func (r *MediaTypeScriptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_script"
}

func (r *MediaTypeScriptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonMediaTypeResourceAttributes()
	attrs["exec_path"] = rschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Path to the script on the Zabbix server.",
	}
	attrs["exec_params"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "Script parameters, one per line.",
	}
	resp.Schema = rschema.Schema{
		MarkdownDescription: "Manages a Zabbix script media type (notification channel via custom script).",
		Attributes:          attrs,
	}
}

func (r *MediaTypeScriptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MediaTypeScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MediaTypeScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := scriptModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.MediaTypeCreate(ctx, r.client, mt)
	if err != nil {
		resp.Diagnostics.AddError("Error creating script media type", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.MediaTypeGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading script media type after create", err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("Script media type not found after create",
			"Zabbix returned no media type immediately after creation.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToScriptModel(ctx, created, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MediaTypeScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading script media type", err.Error())
		return
	}
	if mt == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(mediaTypeToScriptModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MediaTypeScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := scriptModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.MediaTypeUpdate(ctx, r.client, mt); err != nil {
		resp.Diagnostics.AddError("Error updating script media type", err.Error())
		return
	}

	updated, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading script media type after update", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("Script media type not found after update",
			"Zabbix returned no media type immediately after update.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToScriptModel(ctx, updated, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MediaTypeScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.MediaTypeDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting script media type", err.Error())
	}
}

func (r *MediaTypeScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func scriptModelToMediaType(ctx context.Context, m MediaTypeScriptModel) (client.MediaType, diag.Diagnostics) {
	mt, diags := mediaTypeBaseFromModel(ctx, m.MediaTypeBaseModel)
	mt.Type = client.MediaTypeTypeScript
	mt.ExecPath = m.ExecPath.ValueString()
	mt.ExecParams = m.ExecParams.ValueString()
	return mt, diags
}

func mediaTypeToScriptModel(ctx context.Context, mt *client.MediaType, m *MediaTypeScriptModel) diag.Diagnostics {
	diags := mediaTypeBaseToModel(ctx, mt, &m.MediaTypeBaseModel)
	m.ExecPath = types.StringValue(mt.ExecPath)
	m.ExecParams = types.StringValue(mt.ExecParams)
	return diags
}
