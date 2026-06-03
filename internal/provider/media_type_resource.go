package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MediaTypeResource{}
var _ resource.ResourceWithImportState = &MediaTypeResource{}
var _ resource.ResourceWithConfigValidators = &MediaTypeResource{}

func NewMediaTypeResource() resource.Resource {
	return &MediaTypeResource{}
}

type MediaTypeResource struct {
	client client.Client
}

// --- Element attr-type maps (needed for types.List / types.ListValueFrom) ---

var (
	webhookParamAttrTypes = map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	}
	msgTemplateAttrTypes = map[string]attr.Type{
		"eventsource": types.Int64Type,
		"recovery":    types.Int64Type,
		"subject":     types.StringType,
		"message":     types.StringType,
	}
)

// --- Model types ---

type EmailSettingsModel struct {
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

type SMSSettingsModel struct {
	GSMModem types.String `tfsdk:"gsm_modem"`
}

type ScriptSettingsModel struct {
	ExecPath   types.String `tfsdk:"exec_path"`
	ExecParams types.String `tfsdk:"exec_params"`
}

type WebhookParameterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type WebhookSettingsModel struct {
	Script        types.String `tfsdk:"script"`
	Timeout       types.String `tfsdk:"timeout"`
	ProcessTags   types.Bool   `tfsdk:"process_tags"`
	ShowEventMenu types.Bool   `tfsdk:"show_event_menu"`
	EventMenuURL  types.String `tfsdk:"event_menu_url"`
	EventMenuName types.String `tfsdk:"event_menu_name"`
	// types.List handles null/unknown during ImportState; element type: WebhookParameterModel
	Parameters types.List `tfsdk:"parameters"`
}

type MessageTemplateModel struct {
	EventSource types.Int64  `tfsdk:"eventsource"`
	Recovery    types.Int64  `tfsdk:"recovery"`
	Subject     types.String `tfsdk:"subject"`
	Message     types.String `tfsdk:"message"`
}

type MediaTypeResourceModel struct {
	ID              types.String          `tfsdk:"id"`
	Name            types.String          `tfsdk:"name"`
	Type            types.String          `tfsdk:"type"`
	Status          types.String          `tfsdk:"status"`
	Description     types.String          `tfsdk:"description"`
	MaxSessions     types.Int64           `tfsdk:"max_sessions"`
	MaxAttempts     types.Int64           `tfsdk:"max_attempts"`
	AttemptInterval types.String          `tfsdk:"attempt_interval"`
	EmailSettings   *EmailSettingsModel   `tfsdk:"email_settings"`
	SMSSettings     *SMSSettingsModel     `tfsdk:"sms_settings"`
	ScriptSettings  *ScriptSettingsModel  `tfsdk:"script_settings"`
	WebhookSettings *WebhookSettingsModel `tfsdk:"webhook_settings"`
	// types.List handles null/unknown during ImportState; element type: MessageTemplateModel
	MessageTemplates types.List `tfsdk:"message_templates"`
}

// --- Metadata / Schema ---

func (r *MediaTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type"
}

func (r *MediaTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix media type (notification channel). " +
			"Exactly one of `email_settings`, `sms_settings`, `script_settings`, or `webhook_settings` " +
			"must be set and must match the `type` attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the media type.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the media type. Must be unique within Zabbix.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Media type channel. One of: `email`, `sms`, `script`, `webhook`.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Whether the media type is active. One of: `enabled`, `disabled`. Defaults to `enabled`.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Description of the media type.",
			},
			"max_sessions": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				MarkdownDescription: "Maximum number of concurrent sessions. Defaults to `1`.",
			},
			"max_attempts": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
				MarkdownDescription: "Maximum number of delivery attempts. Defaults to `3`.",
			},
			"attempt_interval": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("10s"),
				MarkdownDescription: "Interval between delivery attempts (e.g. `10s`, `1m`). Defaults to `10s`.",
			},
			"email_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Email-specific settings. Required when `type` is `email`.",
				Attributes: map[string]schema.Attribute{
					"smtp_server": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "SMTP server address.",
					},
					"smtp_port": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(25),
						MarkdownDescription: "SMTP server port. Defaults to `25`.",
					},
					"smtp_helo": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SMTP HELO/EHLO hostname.",
					},
					"smtp_email": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "From address for outgoing email.",
					},
					"smtp_security": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("none"),
						MarkdownDescription: "SMTP connection security. One of: `none`, `starttls`, `ssl_tls`. Defaults to `none`.",
					},
					"smtp_authentication": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("none"),
						MarkdownDescription: "SMTP authentication method. One of: `none`, `normal_password`. Defaults to `none`.",
					},
					"username": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SMTP authentication username.",
					},
					"password": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SMTP authentication password. Sensitive; not returned by the API after creation.",
					},
					"content_type": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("html"),
						MarkdownDescription: "Email content type. One of: `text`, `html`. Defaults to `html`.",
					},
				},
			},
			"sms_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "SMS-specific settings. Required when `type` is `sms`.",
				Attributes: map[string]schema.Attribute{
					"gsm_modem": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Serial device path of the GSM modem (e.g. `/dev/ttyS0`).",
					},
				},
			},
			"script_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Script-specific settings. Required when `type` is `script`.",
				Attributes: map[string]schema.Attribute{
					"exec_path": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Path to the script on the Zabbix server.",
					},
					"exec_params": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "Script parameters, one per line.",
					},
				},
			},
			"webhook_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Webhook-specific settings. Required when `type` is `webhook`.",
				Attributes: map[string]schema.Attribute{
					"script": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "JavaScript body of the webhook.",
					},
					"timeout": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("30s"),
						MarkdownDescription: "Script execution timeout (e.g. `30s`). Defaults to `30s`.",
					},
					"process_tags": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether to add event tags from webhook response. Defaults to `false`.",
					},
					"show_event_menu": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Whether to add a link to the event menu. Defaults to `false`.",
					},
					"event_menu_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "URL for the event menu entry. Used when `show_event_menu` is `true`.",
					},
					"event_menu_name": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "Label for the event menu entry. Used when `show_event_menu` is `true`.",
					},
					"parameters": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Key/value pairs passed to the webhook script.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Parameter name.",
								},
								"value": schema.StringAttribute{
									Required:            true,
									Sensitive:           true,
									MarkdownDescription: "Parameter value. Sensitive.",
								},
							},
						},
					},
				},
			},
			"message_templates": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Per-event-source notification templates. If unset, Zabbix defaults are used.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"eventsource": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Event source: 0=triggers, 1=discovery, 2=autoregistration, 3=internal, 4=services.",
						},
						"recovery": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Recovery: 0=operations, 1=recovery, 2=update.",
						},
						"subject": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Message subject.",
						},
						"message": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Message body.",
						},
					},
				},
			},
		},
	}
}

// --- Plan-time validator ---

type mediaTypeSettingsValidator struct{}

func (v mediaTypeSettingsValidator) Description(_ context.Context) string {
	return "Validates that exactly the settings block matching `type` is set."
}

func (v mediaTypeSettingsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v mediaTypeSettingsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data MediaTypeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Type.IsUnknown() || data.Type.IsNull() {
		return
	}

	typeName := data.Type.ValueString()
	emailSet := data.EmailSettings != nil
	smsSet := data.SMSSettings != nil
	scriptSet := data.ScriptSettings != nil
	webhookSet := data.WebhookSettings != nil

	required := map[string]bool{
		"email":   emailSet,
		"sms":     smsSet,
		"script":  scriptSet,
		"webhook": webhookSet,
	}

	if _, valid := required[typeName]; !valid {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Invalid media type",
			fmt.Sprintf("Unknown type %q; must be one of: email, sms, script, webhook.", typeName),
		)
		return
	}

	if !required[typeName] {
		resp.Diagnostics.AddAttributeError(
			path.Root(typeName+"_settings"),
			fmt.Sprintf("Missing %s_settings", typeName),
			fmt.Sprintf("%s_settings is required when type is %q.", typeName, typeName),
		)
	}

	for other, set := range required {
		if other != typeName && set {
			resp.Diagnostics.AddAttributeError(
				path.Root(other+"_settings"),
				fmt.Sprintf("Unexpected %s_settings", other),
				fmt.Sprintf("%s_settings must not be set when type is %q.", other, typeName),
			)
		}
	}
}

func (r *MediaTypeResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{mediaTypeSettingsValidator{}}
}

// --- Configure ---

func (r *MediaTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

// --- CRUD ---

func (r *MediaTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MediaTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := modelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.MediaTypeCreate(ctx, r.client, mt)
	if err != nil {
		resp.Diagnostics.AddError("Error creating media type", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.MediaTypeGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading media type after create", err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("Media type not found after create", "Zabbix returned no media type immediately after creation.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToModel(ctx, created, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MediaTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading media type", err.Error())
		return
	}
	if mt == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(mediaTypeToModel(ctx, mt, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MediaTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MediaTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(preserveSensitiveState(ctx, &data, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mt, diags := modelToMediaType(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.MediaTypeUpdate(ctx, r.client, mt); err != nil {
		resp.Diagnostics.AddError("Error updating media type", err.Error())
		return
	}

	updated, err := client.MediaTypeGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading media type after update", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("Media type not found after update", "Zabbix returned no media type immediately after update.")
		return
	}
	resp.Diagnostics.Append(mediaTypeToModel(ctx, updated, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MediaTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MediaTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.MediaTypeDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting media type", err.Error())
	}
}

func (r *MediaTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Conversion helpers ---

var (
	mediaTypeTypeMap = map[string]int{
		"email":   client.MediaTypeTypeEmail,
		"sms":     client.MediaTypeTypeSMS,
		"script":  client.MediaTypeTypeScript,
		"webhook": client.MediaTypeTypeWebhook,
	}
	mediaTypeTypeReverseMap = map[int]string{
		client.MediaTypeTypeEmail:   "email",
		client.MediaTypeTypeSMS:     "sms",
		client.MediaTypeTypeScript:  "script",
		client.MediaTypeTypeWebhook: "webhook",
	}

	mediaTypeStatusMap = map[string]int{
		"enabled":  client.MediaTypeStatusEnabled,
		"disabled": client.MediaTypeStatusDisabled,
	}
	mediaTypeStatusReverseMap = map[int]string{
		client.MediaTypeStatusEnabled:  "enabled",
		client.MediaTypeStatusDisabled: "disabled",
	}

	smtpSecurityMap = map[string]int{
		"none":     0,
		"starttls": 1,
		"ssl_tls":  2,
	}
	smtpSecurityReverseMap = map[int]string{
		0: "none",
		1: "starttls",
		2: "ssl_tls",
	}

	smtpAuthMap = map[string]int{
		"none":            0,
		"normal_password": 1,
	}
	smtpAuthReverseMap = map[int]string{
		0: "none",
		1: "normal_password",
	}

	contentTypeMap = map[string]int{
		"text": 0,
		"html": 1,
	}
	contentTypeReverseMap = map[int]string{
		0: "text",
		1: "html",
	}
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

func modelToMediaType(ctx context.Context, m MediaTypeResourceModel) (client.MediaType, diag.Diagnostics) {
	var diags diag.Diagnostics
	mt := client.MediaType{
		ID:              m.ID.ValueString(),
		Name:            m.Name.ValueString(),
		Type:            mediaTypeTypeMap[m.Type.ValueString()],
		Status:          mediaTypeStatusMap[m.Status.ValueString()],
		Description:     m.Description.ValueString(),
		MaxSessions:     int(m.MaxSessions.ValueInt64()),
		MaxAttempts:     int(m.MaxAttempts.ValueInt64()),
		AttemptInterval: m.AttemptInterval.ValueString(),
	}

	if m.EmailSettings != nil {
		es := m.EmailSettings
		mt.SMTPServer = es.SMTPServer.ValueString()
		mt.SMTPPort = int(es.SMTPPort.ValueInt64())
		mt.SMTPHelo = es.SMTPHelo.ValueString()
		mt.SMTPEmail = es.SMTPEmail.ValueString()
		mt.SMTPSecurity = smtpSecurityMap[es.SMTPSecurity.ValueString()]
		mt.SMTPAuthentication = smtpAuthMap[es.SMTPAuthentication.ValueString()]
		mt.Username = es.Username.ValueString()
		mt.Passwd = es.Password.ValueString()
		mt.ContentType = contentTypeMap[es.ContentType.ValueString()]
	}

	if m.SMSSettings != nil {
		mt.GSMModem = m.SMSSettings.GSMModem.ValueString()
	}

	if m.ScriptSettings != nil {
		mt.ExecPath = m.ScriptSettings.ExecPath.ValueString()
		mt.ExecParams = m.ScriptSettings.ExecParams.ValueString()
	}

	if m.WebhookSettings != nil {
		ws := m.WebhookSettings
		mt.Script = ws.Script.ValueString()
		mt.Timeout = ws.Timeout.ValueString()
		mt.ProcessTags = boolToInt(ws.ProcessTags.ValueBool())
		mt.ShowEventMenu = boolToInt(ws.ShowEventMenu.ValueBool())
		mt.EventMenuURL = ws.EventMenuURL.ValueString()
		mt.EventMenuName = ws.EventMenuName.ValueString()
		if !ws.Parameters.IsNull() && !ws.Parameters.IsUnknown() {
			var paramModels []WebhookParameterModel
			diags.Append(ws.Parameters.ElementsAs(ctx, &paramModels, false)...)
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
	}

	if !m.MessageTemplates.IsNull() && !m.MessageTemplates.IsUnknown() {
		var tmplModels []MessageTemplateModel
		diags.Append(m.MessageTemplates.ElementsAs(ctx, &tmplModels, false)...)
		if !diags.HasError() {
			tmpl := make([]client.MessageTemplate, len(tmplModels))
			for i, t := range tmplModels {
				tmpl[i] = client.MessageTemplate{
					EventSource: int(t.EventSource.ValueInt64()),
					Recovery:    int(t.Recovery.ValueInt64()),
					Subject:     t.Subject.ValueString(),
					Message:     t.Message.ValueString(),
				}
			}
			mt.MessageTemplates = tmpl
		}
	}

	return mt, diags
}

// mediaTypeToModel updates the model in-place from an API response.
// Sensitive fields (password, webhook parameter values) are not overwritten —
// the caller must preserve them from prior state since the API does not return them.
func mediaTypeToModel(ctx context.Context, mt *client.MediaType, m *MediaTypeResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Name = types.StringValue(mt.Name)
	m.Type = types.StringValue(mediaTypeTypeReverseMap[mt.Type])
	m.Status = types.StringValue(mediaTypeStatusReverseMap[mt.Status])
	m.Description = types.StringValue(mt.Description)
	m.MaxSessions = types.Int64Value(int64(mt.MaxSessions))
	m.MaxAttempts = types.Int64Value(int64(mt.MaxAttempts))
	m.AttemptInterval = types.StringValue(mt.AttemptInterval)

	m.EmailSettings = nil
	m.SMSSettings = nil
	m.ScriptSettings = nil
	m.WebhookSettings = nil

	switch mt.Type {
	case client.MediaTypeTypeEmail:
		m.EmailSettings = &EmailSettingsModel{
			SMTPServer:         types.StringValue(mt.SMTPServer),
			SMTPPort:           types.Int64Value(int64(mt.SMTPPort)),
			SMTPHelo:           types.StringValue(mt.SMTPHelo),
			SMTPEmail:          types.StringValue(mt.SMTPEmail),
			SMTPSecurity:       types.StringValue(smtpSecurityReverseMap[mt.SMTPSecurity]),
			SMTPAuthentication: types.StringValue(smtpAuthReverseMap[mt.SMTPAuthentication]),
			Username:           types.StringValue(mt.Username),
			Password:           types.StringValue(""), // API does not return passwd
			ContentType:        types.StringValue(contentTypeReverseMap[mt.ContentType]),
		}
	case client.MediaTypeTypeSMS:
		m.SMSSettings = &SMSSettingsModel{
			GSMModem: types.StringValue(mt.GSMModem),
		}
	case client.MediaTypeTypeScript:
		m.ScriptSettings = &ScriptSettingsModel{
			ExecPath:   types.StringValue(mt.ExecPath),
			ExecParams: types.StringValue(mt.ExecParams),
		}
	case client.MediaTypeTypeWebhook:
		ws := &WebhookSettingsModel{
			Script:        types.StringValue(mt.Script),
			Timeout:       types.StringValue(mt.Timeout),
			ProcessTags:   types.BoolValue(intToBool(mt.ProcessTags)),
			ShowEventMenu: types.BoolValue(intToBool(mt.ShowEventMenu)),
			EventMenuURL:  types.StringValue(mt.EventMenuURL),
			EventMenuName: types.StringValue(mt.EventMenuName),
		}
		// Webhook parameters: API does NOT return sensitive values.
		// preserveSensitiveState restores prior values by name after Read.
		paramModels := make([]WebhookParameterModel, len(mt.Parameters))
		for i, p := range mt.Parameters {
			paramModels[i] = WebhookParameterModel{
				Name:  types.StringValue(p.Name),
				Value: types.StringValue(""), // sensitive — not echoed by API
			}
		}
		var d diag.Diagnostics
		ws.Parameters, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: webhookParamAttrTypes}, paramModels)
		diags.Append(d...)
		m.WebhookSettings = ws
	}

	// Message templates: update only when previously tracked (non-null in state).
	// This prevents drift when the user has not declared message_templates.
	if !m.MessageTemplates.IsNull() {
		if len(mt.MessageTemplates) > 0 {
			tmplModels := make([]MessageTemplateModel, len(mt.MessageTemplates))
			for i, t := range mt.MessageTemplates {
				tmplModels[i] = MessageTemplateModel{
					EventSource: types.Int64Value(int64(t.EventSource)),
					Recovery:    types.Int64Value(int64(t.Recovery)),
					Subject:     types.StringValue(t.Subject),
					Message:     types.StringValue(t.Message),
				}
			}
			var d diag.Diagnostics
			m.MessageTemplates, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: msgTemplateAttrTypes}, tmplModels)
			diags.Append(d...)
		} else {
			m.MessageTemplates = types.ListValueMust(types.ObjectType{AttrTypes: msgTemplateAttrTypes}, []attr.Value{})
		}
	}

	return diags
}

// preserveSensitiveState copies sensitive values from the prior state into the
// plan model so they survive the Read-after-write cycle (API does not echo them).
func preserveSensitiveState(ctx context.Context, plan, state *MediaTypeResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if plan.EmailSettings != nil && state.EmailSettings != nil {
		plan.EmailSettings.Password = state.EmailSettings.Password
	}

	if plan.WebhookSettings != nil && state.WebhookSettings != nil {
		planParams := plan.WebhookSettings.Parameters
		stateParams := state.WebhookSettings.Parameters
		if !planParams.IsNull() && !planParams.IsUnknown() &&
			!stateParams.IsNull() && !stateParams.IsUnknown() {
			var planParamModels, stateParamModels []WebhookParameterModel
			diags.Append(planParams.ElementsAs(ctx, &planParamModels, false)...)
			diags.Append(stateParams.ElementsAs(ctx, &stateParamModels, false)...)
			if diags.HasError() {
				return diags
			}

			stateValues := make(map[string]types.String, len(stateParamModels))
			for _, sp := range stateParamModels {
				stateValues[sp.Name.ValueString()] = sp.Value
			}
			for i, pp := range planParamModels {
				if sv, ok := stateValues[pp.Name.ValueString()]; ok {
					planParamModels[i].Value = sv
				}
			}
			var d diag.Diagnostics
			plan.WebhookSettings.Parameters, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: webhookParamAttrTypes}, planParamModels)
			diags.Append(d...)
		}
	}

	return diags
}
