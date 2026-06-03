package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MediaTypeEmailResource{}
var _ resource.ResourceWithImportState = &MediaTypeEmailResource{}

func NewMediaTypeEmailResource() resource.Resource {
	return &MediaTypeEmailResource{}
}

type MediaTypeEmailResource struct {
	client client.Client
}

type MediaTypeEmailModel struct {
	MediaTypeBaseModel
	SMTPServer         types.String `tfsdk:"smtp_server"`
	SMTPPort           types.Int64  `tfsdk:"smtp_port"`
	SMTPHelo           types.String `tfsdk:"smtp_helo"`
	SMTPEmail          types.String `tfsdk:"smtp_email"`
	SMTPSecurity       types.String `tfsdk:"smtp_security"`
	SMTPAuthentication types.String `tfsdk:"smtp_authentication"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	ContentType        types.String `tfsdk:"content_type"`
}

var (
	smtpSecurityMap = map[string]int{
		"none": 0, "starttls": 1, "ssl_tls": 2,
	}
	smtpSecurityReverseMap = map[int]string{
		0: "none", 1: "starttls", 2: "ssl_tls",
	}
	smtpAuthMap = map[string]int{
		"none": 0, "normal_password": 1,
	}
	smtpAuthReverseMap = map[int]string{
		0: "none", 1: "normal_password",
	}
	contentTypeMap = map[string]int{
		"text": 0, "html": 1,
	}
	contentTypeReverseMap = map[int]string{
		0: "text", 1: "html",
	}
)

func (r *MediaTypeEmailResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type_email"
}

func (r *MediaTypeEmailResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonMediaTypeResourceAttributes()
	attrs["smtp_server"] = rschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "SMTP server address.",
	}
	attrs["smtp_port"] = rschema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(25),
		MarkdownDescription: "SMTP server port. Defaults to `25`.",
	}
	attrs["smtp_helo"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "SMTP HELO/EHLO hostname.",
	}
	attrs["smtp_email"] = rschema.StringAttribute{
		Required:            true,
		MarkdownDescription: "From address for outgoing email.",
	}
	attrs["smtp_security"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("none"),
		MarkdownDescription: "SMTP connection security. One of: `none`, `starttls`, `ssl_tls`. Defaults to `none`.",
	}
	attrs["smtp_authentication"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("none"),
		MarkdownDescription: "SMTP authentication method. One of: `none`, `normal_password`. Defaults to `none`.",
	}
	attrs["username"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "SMTP authentication username.",
	}
	attrs["password"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		Default:             stringdefault.StaticString(""),
		MarkdownDescription: "SMTP authentication password. Sensitive; not returned by the API after creation.",
	}
	attrs["content_type"] = rschema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString("html"),
		MarkdownDescription: "Email content type. One of: `text`, `html`. Defaults to `html`.",
	}
	resp.Schema = rschema.Schema{
		MarkdownDescription: "Manages a Zabbix email media type (notification channel via SMTP).",
		Attributes:          attrs,
	}
}

func (r *MediaTypeEmailResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MediaTypeEmailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MediaTypeEmailModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save password before the read-after-write clears it (API does not echo passwd).
	savedPassword := data.Password

	mt, diags := emailModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.MediaTypeCreate(ctx, r.client, mt)
	if err != nil {
		resp.Diagnostics.AddError("Error creating email media type", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.MediaTypeGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading email media type after create", err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("Email media type not found after create",
			"Zabbix returned no media type immediately after creation.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToEmailModel(ctx, created, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Password = savedPassword

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeEmailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MediaTypeEmailModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading email media type", err.Error())
		return
	}
	if mt == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	savedPassword := data.Password
	resp.Diagnostics.Append(mediaTypeToEmailModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Password = savedPassword

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeEmailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MediaTypeEmailModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MediaTypeEmailModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve password from state; plan has the user-declared value but we need
	// the previously stored value if the user hasn't changed it.
	if data.Password.IsNull() || data.Password.ValueString() == "" {
		data.Password = state.Password
	}
	savedPassword := data.Password

	mt, diags := emailModelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.MediaTypeUpdate(ctx, r.client, mt); err != nil {
		resp.Diagnostics.AddError("Error updating email media type", err.Error())
		return
	}

	updated, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading email media type after update", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("Email media type not found after update",
			"Zabbix returned no media type immediately after update.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToEmailModel(ctx, updated, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Password = savedPassword

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeEmailResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MediaTypeEmailModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.MediaTypeDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting email media type", err.Error())
	}
}

func (r *MediaTypeEmailResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Conversion helpers ---

func emailModelToMediaType(ctx context.Context, m MediaTypeEmailModel) (client.MediaType, diag.Diagnostics) {
	mt, diags := mediaTypeBaseFromModel(ctx, m.MediaTypeBaseModel)
	mt.Type = client.MediaTypeTypeEmail
	mt.SMTPServer = m.SMTPServer.ValueString()
	mt.SMTPPort = int(m.SMTPPort.ValueInt64())
	mt.SMTPHelo = m.SMTPHelo.ValueString()
	mt.SMTPEmail = m.SMTPEmail.ValueString()
	mt.SMTPSecurity = smtpSecurityMap[m.SMTPSecurity.ValueString()]
	mt.SMTPAuthentication = smtpAuthMap[m.SMTPAuthentication.ValueString()]
	mt.Username = m.Username.ValueString()
	mt.Passwd = m.Password.ValueString()
	mt.ContentType = contentTypeMap[m.ContentType.ValueString()]
	return mt, diags
}

// mediaTypeToEmailModel populates the email model from an API response.
// Password is set to "" — callers must restore it from prior state or plan since
// the API does not return it.
func mediaTypeToEmailModel(ctx context.Context, mt *client.MediaType, m *MediaTypeEmailModel) diag.Diagnostics {
	diags := mediaTypeBaseToModel(ctx, mt, &m.MediaTypeBaseModel)
	m.SMTPServer = types.StringValue(mt.SMTPServer)
	m.SMTPPort = types.Int64Value(int64(mt.SMTPPort))
	m.SMTPHelo = types.StringValue(mt.SMTPHelo)
	m.SMTPEmail = types.StringValue(mt.SMTPEmail)
	m.SMTPSecurity = types.StringValue(smtpSecurityReverseMap[mt.SMTPSecurity])
	m.SMTPAuthentication = types.StringValue(smtpAuthReverseMap[mt.SMTPAuthentication])
	m.Username = types.StringValue(mt.Username)
	m.Password = types.StringValue("") // API does not return passwd
	m.ContentType = types.StringValue(contentTypeReverseMap[mt.ContentType])
	return diags
}
