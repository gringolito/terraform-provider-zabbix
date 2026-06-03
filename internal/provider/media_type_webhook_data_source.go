package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeWebhookDataSource{}

func NewMediaTypeWebhookDataSource() datasource.DataSource {
	return &MediaTypeWebhookDataSource{}
}

type MediaTypeWebhookDataSource struct {
	client client.Client
}

var webhookParamDSAttrTypes = map[string]attr.Type{
	"name": types.StringType,
}

// WebhookParameterDataSourceModel holds only the parameter name for data source reads.
// Parameter values are never returned by the Zabbix API.
type WebhookParameterDataSourceModel struct {
	Name types.String `tfsdk:"name"`
}

// MediaTypeWebhookDataSourceModel is the data source model for the webhook media type.
type MediaTypeWebhookDataSourceModel struct {
	MediaTypeBaseModel
	Script        types.String `tfsdk:"script"`
	Timeout       types.String `tfsdk:"timeout"`
	ProcessTags   types.Bool   `tfsdk:"process_tags"`
	ShowEventMenu types.Bool   `tfsdk:"show_event_menu"`
	EventMenuURL  types.String `tfsdk:"event_menu_url"`
	EventMenuName types.String `tfsdk:"event_menu_name"`
	Parameters    types.List   `tfsdk:"parameters"`
}

func (d *MediaTypeWebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_webhook"
}

func (d *MediaTypeWebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := commonMediaTypeDataSourceAttributes()
	attrs["script"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "JavaScript body of the webhook."}
	attrs["timeout"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "Script execution timeout."}
	attrs["process_tags"] = dschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether to add event tags from webhook response."}
	attrs["show_event_menu"] = dschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether to add a link to the event menu."}
	attrs["event_menu_url"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "URL for the event menu entry."}
	attrs["event_menu_name"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "Label for the event menu entry."}
	attrs["parameters"] = dschema.ListNestedAttribute{
		Computed:            true,
		MarkdownDescription: "Key names of the parameters passed to the webhook script. Parameter values are not returned by the API.",
		NestedObject: dschema.NestedAttributeObject{
			Attributes: map[string]dschema.Attribute{
				"name": dschema.StringAttribute{Computed: true, MarkdownDescription: "Parameter name."},
			},
		},
	}
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Fetches a Zabbix webhook media type by `id` or `name`. Exactly one of `id` or `name` must be provided.",
		Attributes:          attrs,
	}
}

func (d *MediaTypeWebhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *MediaTypeWebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MediaTypeWebhookDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := lookupMediaType(ctx, d.client, data.ID, data.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.MessageTemplates = types.ListValueMust(types.ObjectType{AttrTypes: msgTemplateAttrTypes}, nil)
	data.ID = types.StringValue(mt.ID)

	resp.Diagnostics.Append(mediaTypeToWebhookDataSourceModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mediaTypeToWebhookDataSourceModel(ctx context.Context, mt *client.MediaType, m *MediaTypeWebhookDataSourceModel) diag.Diagnostics {
	diags := mediaTypeBaseToModel(ctx, mt, &m.MediaTypeBaseModel)
	m.Script = types.StringValue(mt.Script)
	m.Timeout = types.StringValue(mt.Timeout)
	m.ProcessTags = types.BoolValue(intToBool(mt.ProcessTags))
	m.ShowEventMenu = types.BoolValue(intToBool(mt.ShowEventMenu))
	m.EventMenuURL = types.StringValue(mt.EventMenuURL)
	m.EventMenuName = types.StringValue(mt.EventMenuName)

	paramModels := make([]WebhookParameterDataSourceModel, len(mt.Parameters))
	for i, p := range mt.Parameters {
		paramModels[i] = WebhookParameterDataSourceModel{
			Name: types.StringValue(p.Name),
		}
	}
	var d diag.Diagnostics
	m.Parameters, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: webhookParamDSAttrTypes}, paramModels)
	diags.Append(d...)
	return diags
}
