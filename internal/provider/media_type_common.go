package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- Shared attr-type maps ---

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

// --- Shared model types ---

// MediaTypeBaseModel holds fields common to all media type resources and data sources.
// Embed this struct (not a pointer) in each type-specific model.
type MediaTypeBaseModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Status          types.String `tfsdk:"status"`
	Description     types.String `tfsdk:"description"`
	MaxSessions     types.Int64  `tfsdk:"max_sessions"`
	MaxAttempts     types.Int64  `tfsdk:"max_attempts"`
	AttemptInterval types.String `tfsdk:"attempt_interval"`
	// types.List handles null/unknown during ImportState; element type: MessageTemplateModel
	MessageTemplates types.List `tfsdk:"message_templates"`
}

type MessageTemplateModel struct {
	EventSource types.Int64  `tfsdk:"eventsource"`
	Recovery    types.Int64  `tfsdk:"recovery"`
	Subject     types.String `tfsdk:"subject"`
	Message     types.String `tfsdk:"message"`
}

type WebhookParameterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// --- Shared status maps ---

var (
	mediaTypeStatusMap = map[string]int{
		"enabled":  client.MediaTypeStatusEnabled,
		"disabled": client.MediaTypeStatusDisabled,
	}
	mediaTypeStatusReverseMap = map[int]string{
		client.MediaTypeStatusEnabled:  "enabled",
		client.MediaTypeStatusDisabled: "disabled",
	}
)

// --- Schema builders ---

// commonMediaTypeResourceAttributes returns schema attributes shared by all media type resources.
// Merge the returned map into the type-specific schema.Attributes.
func commonMediaTypeResourceAttributes() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Unique identifier of the media type.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": rschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Display name of the media type. Must be unique within Zabbix.",
		},
		"status": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("enabled"),
			MarkdownDescription: "Whether the media type is active. One of: `enabled`, `disabled`. Defaults to `enabled`.",
		},
		"description": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Description of the media type.",
		},
		"max_sessions": rschema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(1),
			MarkdownDescription: "Maximum number of concurrent sessions. Defaults to `1`.",
		},
		"max_attempts": rschema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(3),
			MarkdownDescription: "Maximum number of delivery attempts. Defaults to `3`.",
		},
		"attempt_interval": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("10s"),
			MarkdownDescription: "Interval between delivery attempts (e.g. `10s`, `1m`). Defaults to `10s`.",
		},
		"message_templates": rschema.ListNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Per-event-source notification templates. If unset, Zabbix defaults are used.",
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"eventsource": rschema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Event source: 0=triggers, 1=discovery, 2=autoregistration, 3=internal, 4=services.",
					},
					"recovery": rschema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Recovery: 0=operations, 1=recovery, 2=update.",
					},
					"subject": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Message subject.",
					},
					"message": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Message body.",
					},
				},
			},
		},
	}
}

// commonMediaTypeDataSourceAttributes returns schema attributes shared by all media type data sources.
// Merge the returned map into the type-specific schema.Attributes.
func commonMediaTypeDataSourceAttributes() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"id": dschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Unique identifier of the media type. One of `id` or `name` must be set.",
		},
		"name": dschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Display name of the media type. One of `id` or `name` must be set.",
		},
		"status": dschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the media type is active: `enabled` or `disabled`.",
		},
		"description": dschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Description of the media type.",
		},
		"max_sessions": dschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Maximum number of concurrent sessions.",
		},
		"max_attempts": dschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Maximum number of delivery attempts.",
		},
		"attempt_interval": dschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Interval between delivery attempts.",
		},
		"message_templates": dschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Per-event-source notification templates.",
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"eventsource": dschema.Int64Attribute{Computed: true, MarkdownDescription: "Event source type."},
					"recovery":    dschema.Int64Attribute{Computed: true, MarkdownDescription: "Recovery type."},
					"subject":     dschema.StringAttribute{Computed: true, MarkdownDescription: "Message subject."},
					"message":     dschema.StringAttribute{Computed: true, MarkdownDescription: "Message body."},
				},
			},
		},
	}
}

// --- Shared converters ---

// mediaTypeBaseFromModel fills common client.MediaType fields from the embedded base model.
func mediaTypeBaseFromModel(ctx context.Context, m MediaTypeBaseModel) (client.MediaType, diag.Diagnostics) {
	var diags diag.Diagnostics
	mt := client.MediaType{
		ID:              m.ID.ValueString(),
		Name:            m.Name.ValueString(),
		Status:          mediaTypeStatusMap[m.Status.ValueString()],
		Description:     m.Description.ValueString(),
		MaxSessions:     int(m.MaxSessions.ValueInt64()),
		MaxAttempts:     int(m.MaxAttempts.ValueInt64()),
		AttemptInterval: m.AttemptInterval.ValueString(),
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

// mediaTypeBaseToModel populates the embedded base model from an API response.
// Message templates are updated only when MessageTemplates is non-null in the model,
// preserving the "untracked if not declared" semantics for resources.
func mediaTypeBaseToModel(ctx context.Context, mt *client.MediaType, m *MediaTypeBaseModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Name = types.StringValue(mt.Name)
	m.Status = types.StringValue(mediaTypeStatusReverseMap[mt.Status])
	m.Description = types.StringValue(mt.Description)
	m.MaxSessions = types.Int64Value(int64(mt.MaxSessions))
	m.MaxAttempts = types.Int64Value(int64(mt.MaxAttempts))
	m.AttemptInterval = types.StringValue(mt.AttemptInterval)

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

// lookupMediaType fetches a media type by id or name. Exactly one must be non-null.
func lookupMediaType(ctx context.Context, c client.Client, id, name types.String) (*client.MediaType, diag.Diagnostics) {
	var diags diag.Diagnostics

	if id.IsNull() && name.IsNull() {
		diags.AddError("Missing lookup key", "Exactly one of `id` or `name` must be set.")
		return nil, diags
	}

	if !id.IsNull() {
		mt, err := client.MediaTypeGet(ctx, c, id.ValueString())
		if err != nil {
			diags.AddError("Error reading media type", err.Error())
			return nil, diags
		}
		if mt == nil {
			diags.AddError("Media type not found", fmt.Sprintf("No media type found with id %q.", id.ValueString()))
			return nil, diags
		}
		return mt, diags
	}

	mts, err := client.MediaTypeGetByName(ctx, c, name.ValueString())
	if err != nil {
		diags.AddError("Error reading media type", err.Error())
		return nil, diags
	}
	switch len(mts) {
	case 0:
		diags.AddError("Media type not found", fmt.Sprintf("No media type found with name %q.", name.ValueString()))
	case 1:
		return &mts[0], diags
	default:
		diags.AddError("Multiple media types found", fmt.Sprintf("Found %d media types with name %q; use `id` to disambiguate.", len(mts), name.ValueString()))
	}
	return nil, diags
}
