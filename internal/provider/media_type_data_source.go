package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MediaTypeDataSource{}

func NewMediaTypeDataSource() datasource.DataSource {
	return &MediaTypeDataSource{}
}

type MediaTypeDataSource struct {
	client client.Client
}

// MediaTypeDataSourceModel reuses the nested model types from the resource.
type MediaTypeDataSourceModel struct {
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
	// types.List handles null/unknown; element type: MessageTemplateModel
	MessageTemplates types.List `tfsdk:"message_templates"`
}

func (d *MediaTypeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_media_type"
}

func (d *MediaTypeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix media type by `id` or `name`. Exactly one of `id` or `name` must be provided.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the media type. One of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display name of the media type. One of `id` or `name` must be set.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Media type channel: `email`, `sms`, `script`, or `webhook`.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the media type is active: `enabled` or `disabled`.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the media type.",
			},
			"max_sessions": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum number of concurrent sessions.",
			},
			"max_attempts": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum number of delivery attempts.",
			},
			"attempt_interval": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Interval between delivery attempts.",
			},
			"email_settings": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Email-specific settings. Populated when `type` is `email`.",
				Attributes: map[string]schema.Attribute{
					"smtp_server":         schema.StringAttribute{Computed: true, MarkdownDescription: "SMTP server address."},
					"smtp_port":           schema.Int64Attribute{Computed: true, MarkdownDescription: "SMTP server port."},
					"smtp_helo":           schema.StringAttribute{Computed: true, MarkdownDescription: "SMTP HELO/EHLO hostname."},
					"smtp_email":          schema.StringAttribute{Computed: true, MarkdownDescription: "From address for outgoing email."},
					"smtp_security":       schema.StringAttribute{Computed: true, MarkdownDescription: "SMTP connection security."},
					"smtp_authentication": schema.StringAttribute{Computed: true, MarkdownDescription: "SMTP authentication method."},
					"username":            schema.StringAttribute{Computed: true, MarkdownDescription: "SMTP authentication username."},
					"password":            schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "SMTP authentication password. Always empty — the API does not return passwords."},
					"content_type":        schema.StringAttribute{Computed: true, MarkdownDescription: "Email content type."},
				},
			},
			"sms_settings": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "SMS-specific settings. Populated when `type` is `sms`.",
				Attributes: map[string]schema.Attribute{
					"gsm_modem": schema.StringAttribute{Computed: true, MarkdownDescription: "Serial device path of the GSM modem."},
				},
			},
			"script_settings": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Script-specific settings. Populated when `type` is `script`.",
				Attributes: map[string]schema.Attribute{
					"exec_path":   schema.StringAttribute{Computed: true, MarkdownDescription: "Path to the script on the Zabbix server."},
					"exec_params": schema.StringAttribute{Computed: true, MarkdownDescription: "Script parameters."},
				},
			},
			"webhook_settings": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Webhook-specific settings. Populated when `type` is `webhook`.",
				Attributes: map[string]schema.Attribute{
					"script":          schema.StringAttribute{Computed: true, MarkdownDescription: "JavaScript body of the webhook."},
					"timeout":         schema.StringAttribute{Computed: true, MarkdownDescription: "Script execution timeout."},
					"process_tags":    schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether to process event tags from the webhook response."},
					"show_event_menu": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether to add a link to the event menu."},
					"event_menu_url":  schema.StringAttribute{Computed: true, MarkdownDescription: "URL for the event menu entry."},
					"event_menu_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Label for the event menu entry."},
					"parameters": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Key/value pairs passed to the webhook script.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name":  schema.StringAttribute{Computed: true, MarkdownDescription: "Parameter name."},
								"value": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Parameter value. Always empty — the API does not return sensitive values."},
							},
						},
					},
				},
			},
			"message_templates": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Per-event-source notification templates.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"eventsource": schema.Int64Attribute{Computed: true, MarkdownDescription: "Event source type."},
						"recovery":    schema.Int64Attribute{Computed: true, MarkdownDescription: "Recovery type."},
						"subject":     schema.StringAttribute{Computed: true, MarkdownDescription: "Message subject."},
						"message":     schema.StringAttribute{Computed: true, MarkdownDescription: "Message body."},
					},
				},
			},
		},
	}
}

func (d *MediaTypeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *MediaTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MediaTypeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Exactly one of `id` or `name` must be set.",
		)
		return
	}

	var mt *client.MediaType

	if !data.ID.IsNull() {
		found, err := client.MediaTypeGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading media type", err.Error())
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Media type not found",
				fmt.Sprintf("No media type found with id %q.", data.ID.ValueString()),
			)
			return
		}
		mt = found
	} else {
		mts, err := client.MediaTypeGetByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading media type", err.Error())
			return
		}
		switch len(mts) {
		case 0:
			resp.Diagnostics.AddError(
				"Media type not found",
				fmt.Sprintf("No media type found with name %q.", data.Name.ValueString()),
			)
			return
		case 1:
			mt = &mts[0]
		default:
			resp.Diagnostics.AddError(
				"Multiple media types found",
				fmt.Sprintf("Found %d media types with name %q; use `id` to disambiguate.", len(mts), data.Name.ValueString()),
			)
			return
		}
	}

	// Convert API response to the data source model.
	// Pre-set MessageTemplates to a non-null empty list so mediaTypeToModel
	// always populates it from the API (data source always tracks all fields).
	rm := &MediaTypeResourceModel{
		ID:               data.ID,
		MessageTemplates: types.ListValueMust(types.ObjectType{AttrTypes: msgTemplateAttrTypes}, nil),
	}
	resp.Diagnostics.Append(mediaTypeToModel(ctx, mt, rm)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(mt.ID)
	data.Name = rm.Name
	data.Type = rm.Type
	data.Status = rm.Status
	data.Description = rm.Description
	data.MaxSessions = rm.MaxSessions
	data.MaxAttempts = rm.MaxAttempts
	data.AttemptInterval = rm.AttemptInterval
	data.EmailSettings = rm.EmailSettings
	data.SMSSettings = rm.SMSSettings
	data.ScriptSettings = rm.ScriptSettings
	data.WebhookSettings = rm.WebhookSettings
	data.MessageTemplates = rm.MessageTemplates

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
