package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeEmailDataSource{}

func NewMediaTypeEmailDataSource() datasource.DataSource {
	return &MediaTypeEmailDataSource{}
}

type MediaTypeEmailDataSource struct {
	client client.Client
}

// MediaTypeEmailDataSourceModel is the data source model for the email media type.
// It does not include password — the API never returns it.
type MediaTypeEmailDataSourceModel struct {
	MediaTypeBaseModel
	SMTPServer         types.String `tfsdk:"smtp_server"`
	SMTPPort           types.Int64  `tfsdk:"smtp_port"`
	SMTPHelo           types.String `tfsdk:"smtp_helo"`
	SMTPEmail          types.String `tfsdk:"smtp_email"`
	SMTPSecurity       types.String `tfsdk:"smtp_security"`
	SMTPAuthentication types.String `tfsdk:"smtp_authentication"`
	Username           types.String `tfsdk:"username"`
	ContentType        types.String `tfsdk:"content_type"`
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
	var data MediaTypeEmailDataSourceModel
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

	resp.Diagnostics.Append(mediaTypeToEmailDataSourceModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mediaTypeToEmailDataSourceModel(ctx context.Context, mt *client.MediaType, m *MediaTypeEmailDataSourceModel) diag.Diagnostics {
	diags := mediaTypeBaseToModel(ctx, mt, &m.MediaTypeBaseModel)
	m.SMTPServer = types.StringValue(mt.SMTPServer)
	m.SMTPPort = types.Int64Value(int64(mt.SMTPPort))
	m.SMTPHelo = types.StringValue(mt.SMTPHelo)
	m.SMTPEmail = types.StringValue(mt.SMTPEmail)
	m.SMTPSecurity = types.StringValue(smtpSecurityReverseMap[mt.SMTPSecurity])
	m.SMTPAuthentication = types.StringValue(smtpAuthReverseMap[mt.SMTPAuthentication])
	m.Username = types.StringValue(mt.Username)
	m.ContentType = types.StringValue(contentTypeReverseMap[mt.ContentType])
	return diags
}
