package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeSMSDataSource{}

func NewMediaTypeSMSDataSource() datasource.DataSource {
	return &MediaTypeSMSDataSource{}
}

type MediaTypeSMSDataSource struct {
	client client.Client
}

func (d *MediaTypeSMSDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_sms"
}

func (d *MediaTypeSMSDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := commonMediaTypeDataSourceAttributes()
	attrs["gsm_modem"] = dschema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Serial device path of the GSM modem.",
	}
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Fetches a Zabbix SMS media type by `id` or `name`. Exactly one of `id` or `name` must be provided.",
		Attributes:          attrs,
	}
}

func (d *MediaTypeSMSDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MediaTypeSMSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MediaTypeSMSModel
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

	resp.Diagnostics.Append(mediaTypeToSMSModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
