package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeEmailDataSource{}

func NewMediaTypeEmailDataSource() datasource.DataSource {
	return &MediaTypeEmailDataSource{}
}

type MediaTypeEmailDataSource struct {
	client client.Client
}

func (d *MediaTypeEmailDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_email"
}

func (d *MediaTypeEmailDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := commonMediaTypeDataSourceAttributes()
	attrs["smtp_server"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "SMTP server address."}
	attrs["smtp_port"] = dschema.Int64Attribute{Computed: true, MarkdownDescription: "SMTP server port."}
	attrs["smtp_helo"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "SMTP HELO/EHLO hostname."}
	attrs["smtp_email"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "From address for outgoing email."}
	attrs["smtp_security"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "SMTP connection security."}
	attrs["smtp_authentication"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "SMTP authentication method."}
	attrs["username"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "SMTP authentication username."}
	attrs["password"] = dschema.StringAttribute{
		Computed:            true,
		Sensitive:           true,
		MarkdownDescription: "SMTP authentication password. Always empty — the API does not return passwords.",
	}
	attrs["content_type"] = dschema.StringAttribute{Computed: true, MarkdownDescription: "Email content type."}
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Fetches a Zabbix email media type by `id` or `name`. Exactly one of `id` or `name` must be provided.",
		Attributes:          attrs,
	}
}

func (d *MediaTypeEmailDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MediaTypeEmailDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MediaTypeEmailModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := lookupMediaType(ctx, d.client, data.ID, data.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pre-set MessageTemplates to non-null so mediaTypeBaseToModel always populates it.
	data.MessageTemplates = types.ListValueMust(types.ObjectType{AttrTypes: msgTemplateAttrTypes}, nil)
	data.ID = types.StringValue(mt.ID)

	resp.Diagnostics.Append(mediaTypeToEmailModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
