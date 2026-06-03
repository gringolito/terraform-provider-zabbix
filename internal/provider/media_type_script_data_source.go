package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeScriptDataSource{}

func NewMediaTypeScriptDataSource() datasource.DataSource {
	return &MediaTypeScriptDataSource{}
}

type MediaTypeScriptDataSource struct {
	client client.Client
}

func (d *MediaTypeScriptDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_script"
}

func (d *MediaTypeScriptDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := commonMediaTypeDataSourceAttributes()
	attrs["exec_path"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "Path to the script on the Zabbix server."}
	attrs["exec_params"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "Script parameters, one per line."}
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Fetches a Zabbix script media type by `id` or `name`. Exactly one of `id` or `name` must be provided.",
		Attributes:          attrs,
	}
}

func (d *MediaTypeScriptDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MediaTypeScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MediaTypeScriptModel
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

	resp.Diagnostics.Append(mediaTypeToScriptModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
