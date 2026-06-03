package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MediaTypeSMSResource{}
var _ resource.ResourceWithImportState = &MediaTypeSMSResource{}

func NewMediaTypeSMSResource() resource.Resource {
	return &MediaTypeSMSResource{}
}

type MediaTypeSMSResource struct {
	client client.Client
}

type MediaTypeSMSModel struct {
	MediaTypeBaseModel
	GSMModem types.String `tfsdk:"gsm_modem"`
}

func (r *MediaTypeSMSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_sms"
}

func (r *MediaTypeSMSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonMediaTypeResourceAttributes()
	attrs["gsm_modem"] = rschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Serial device path of the GSM modem (e.g. `/dev/ttyS0`).",
	}
	resp.Schema = rschema.Schema{
		MarkdownDescription: "Manages a Zabbix SMS media type (notification channel via GSM modem).",
		Attributes:          attrs,
	}
}

func (r *MediaTypeSMSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MediaTypeSMSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MediaTypeSMSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := smsModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.MediaTypeCreate(ctx, r.client, mt)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SMS media type", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.MediaTypeGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading SMS media type after create", err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("SMS media type not found after create",
			"Zabbix returned no media type immediately after creation.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToSMSModel(ctx, created, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeSMSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MediaTypeSMSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SMS media type", err.Error())
		return
	}
	if mt == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(mediaTypeToSMSModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeSMSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MediaTypeSMSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := smsModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.MediaTypeUpdate(ctx, r.client, mt); err != nil {
		resp.Diagnostics.AddError("Error updating SMS media type", err.Error())
		return
	}

	updated, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SMS media type after update", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("SMS media type not found after update",
			"Zabbix returned no media type immediately after update.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToSMSModel(ctx, updated, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeSMSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MediaTypeSMSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.MediaTypeDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting SMS media type", err.Error())
	}
}

func (r *MediaTypeSMSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func smsModelToMediaType(ctx context.Context, m MediaTypeSMSModel) (client.MediaType, diag.Diagnostics) {
	mt, diags := mediaTypeBaseFromModel(ctx, m.MediaTypeBaseModel)
	mt.Type = client.MediaTypeTypeSMS
	mt.GSMModem = m.GSMModem.ValueString()
	return mt, diags
}

func mediaTypeToSMSModel(ctx context.Context, mt *client.MediaType, m *MediaTypeSMSModel) diag.Diagnostics {
	diags := mediaTypeBaseToModel(ctx, mt, &m.MediaTypeBaseModel)
	m.GSMModem = types.StringValue(mt.GSMModem)
	return diags
}
