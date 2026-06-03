package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeWebhookDataSource{}

func NewMediaTypeWebhookDataSource() datasource.DataSource {
	return &MediaTypeWebhookDataSource{}
}

type MediaTypeWebhookDataSource struct {
	client client.Client
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
		MarkdownDescription: "Key/value pairs passed to the webhook script. Parameter values are always empty — the API does not return sensitive values.",
		NestedObject: dschema.NestedAttributeObject{
			Attributes: map[string]dschema.Attribute{
				"name":  dschema.StringAttribute{Computed: true, MarkdownDescription: "Parameter name."},
				"value": dschema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Parameter value. Always empty — the API does not return sensitive values."},
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
	var data MediaTypeWebhookModel
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

	resp.Diagnostics.Append(mediaTypeToWebhookModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
