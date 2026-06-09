package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	provisionGroupAttrTypes = map[string]attr.Type{
		"name":           types.StringType,
		"role_id":        types.StringType,
		"user_group_ids": types.SetType{ElemType: types.StringType},
	}
	provisionMediaAttrTypes = map[string]attr.Type{
		"name":          types.StringType,
		"media_type_id": types.StringType,
		"attribute":     types.StringType,
		"active":        types.StringType,
		"severity":      types.Int64Type,
		"period":        types.StringType,
	}
	provisionStatusMap = map[string]int64{
		"disabled": 0,
		"enabled":  1,
	}
	provisionStatusReverseMap = map[int64]string{
		0: "disabled",
		1: "enabled",
	}
	provisionMediaActiveMap = map[string]int64{
		"enabled":  0,
		"disabled": 1,
	}
	provisionMediaActiveReverseMap = map[int64]string{
		0: "enabled",
		1: "disabled",
	}
	udEnabledDisabledMap = map[string]int64{
		"disabled": 0,
		"enabled":  1,
	}
	udEnabledDisabledReverseMap = map[int64]string{
		0: "disabled",
		1: "enabled",
	}
)

// UserDirectoryBaseModel holds fields common to all user directory resources and data sources.
// Embed this struct (not a pointer) in each type-specific model.
type UserDirectoryBaseModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ProvisionStatus types.String `tfsdk:"provision_status"`
	GroupName       types.String `tfsdk:"group_name"`
	UserUsername    types.String `tfsdk:"user_username"`
	UserLastname    types.String `tfsdk:"user_lastname"`
	ProvisionGroups types.List   `tfsdk:"provision_groups"`
	ProvisionMedia  types.List   `tfsdk:"provision_media"`
}

type ProvisionGroupModel struct {
	Name       types.String `tfsdk:"name"`
	RoleID     types.String `tfsdk:"role_id"`
	UserGroups types.Set    `tfsdk:"user_group_ids"`
}

type ProvisionMediaModel struct {
	Name        types.String `tfsdk:"name"`
	MediaTypeID types.String `tfsdk:"media_type_id"`
	Attribute   types.String `tfsdk:"attribute"`
	Active      types.String `tfsdk:"active"`
	Severity    types.Int64  `tfsdk:"severity"`
	Period      types.String `tfsdk:"period"`
}

func commonUserDirectoryResourceAttributes() map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Unique identifier of the user directory.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": rschema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Display name of the user directory. Must be unique within Zabbix.",
		},
		"description": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Description of the user directory.",
		},
		"provision_status": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("disabled"),
			MarkdownDescription: "Whether JIT provisioning is enabled. One of: `enabled`, `disabled`. Defaults to `disabled`.",
			Validators: []validator.String{
				stringvalidator.OneOf("enabled", "disabled"),
			},
		},
		"group_name": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Name of the IdP attribute that carries group membership.",
		},
		"user_username": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "IdP attribute mapped to the Zabbix user first name.",
		},
		"user_lastname": rschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "IdP attribute mapped to the Zabbix user last name.",
		},
		"provision_groups": rschema.ListNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "IdP group to Zabbix role and user groups mappings for JIT provisioning.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"name": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Name of the IdP group. Use `*` as a wildcard for all groups.",
					},
					"role_id": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "ID of the Zabbix role to assign.",
					},
					"user_group_ids": rschema.SetAttribute{
						Required:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Set of Zabbix user group IDs to assign.",
					},
				},
			},
		},
		"provision_media": rschema.ListNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "IdP attribute to Zabbix media type mappings for JIT provisioning.",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"name": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Name of the provision media entry.",
					},
					"media_type_id": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "ID of the Zabbix media type.",
					},
					"attribute": rschema.StringAttribute{
						Required:            true,
						MarkdownDescription: "IdP attribute value to use as the media send-to address.",
					},
					"active": rschema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("enabled"),
						MarkdownDescription: "Whether the media is active. One of: `enabled`, `disabled`. Defaults to `enabled`.",
						Validators: []validator.String{
							stringvalidator.OneOf("enabled", "disabled"),
						},
					},
					"severity": rschema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(63),
						MarkdownDescription: "Severity bitmask (0-63). Defaults to `63` (all severities).",
						Validators: []validator.Int64{
							int64validator.Between(0, 63),
						},
					},
					"period": rschema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("1-7,00:00-24:00"),
						MarkdownDescription: "Active time period. Defaults to `1-7,00:00-24:00`.",
					},
				},
			},
		},
	}
}

func commonUserDirectoryDataSourceAttributes() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"id": dschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Unique identifier of the user directory. One of `id` or `name` must be set.",
		},
		"name": dschema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Display name of the user directory. One of `id` or `name` must be set.",
		},
		"description":      dschema.StringAttribute{Computed: true, MarkdownDescription: "Description of the user directory."},
		"provision_status": dschema.StringAttribute{Computed: true, MarkdownDescription: "Whether JIT provisioning is enabled: `enabled` or `disabled`."},
		"group_name":       dschema.StringAttribute{Computed: true, MarkdownDescription: "Name of the IdP attribute that carries group membership."},
		"user_username":    dschema.StringAttribute{Computed: true, MarkdownDescription: "IdP attribute mapped to the Zabbix user first name."},
		"user_lastname":    dschema.StringAttribute{Computed: true, MarkdownDescription: "IdP attribute mapped to the Zabbix user last name."},
		"provision_groups": dschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "IdP group to Zabbix role and user groups mappings.",
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"name":           dschema.StringAttribute{Computed: true, MarkdownDescription: "Name of the IdP group."},
					"role_id":        dschema.StringAttribute{Computed: true, MarkdownDescription: "ID of the Zabbix role."},
					"user_group_ids": dschema.SetAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Set of Zabbix user group IDs."},
				},
			},
		},
		"provision_media": dschema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "IdP attribute to Zabbix media type mappings.",
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"name":          dschema.StringAttribute{Computed: true, MarkdownDescription: "Name of the provision media entry."},
					"media_type_id": dschema.StringAttribute{Computed: true, MarkdownDescription: "ID of the Zabbix media type."},
					"attribute":     dschema.StringAttribute{Computed: true, MarkdownDescription: "IdP attribute value."},
					"active":        dschema.StringAttribute{Computed: true, MarkdownDescription: "Whether the media is active: `enabled` or `disabled`."},
					"severity":      dschema.Int64Attribute{Computed: true, MarkdownDescription: "Severity bitmask."},
					"period":        dschema.StringAttribute{Computed: true, MarkdownDescription: "Active time period."},
				},
			},
		},
	}
}

func userDirectoryBaseFromModel(ctx context.Context, m *UserDirectoryBaseModel) (client.UserDirectory, diag.Diagnostics) {
	var diags diag.Diagnostics
	ud := client.UserDirectory{
		Name:            m.Name.ValueString(),
		Description:     m.Description.ValueString(),
		ProvisionStatus: provisionStatusMap[m.ProvisionStatus.ValueString()],
		GroupName:       m.GroupName.ValueString(),
		UserUsername:    m.UserUsername.ValueString(),
		UserLastname:    m.UserLastname.ValueString(),
	}

	if !m.ProvisionGroups.IsNull() && !m.ProvisionGroups.IsUnknown() {
		var groupModels []ProvisionGroupModel
		diags.Append(m.ProvisionGroups.ElementsAs(ctx, &groupModels, false)...)
		if diags.HasError() {
			return ud, diags
		}
		groups := make([]client.ProvisionGroup, len(groupModels))
		for i, gm := range groupModels {
			var ugIDs []string
			diags.Append(gm.UserGroups.ElementsAs(ctx, &ugIDs, false)...)
			if diags.HasError() {
				return ud, diags
			}
			ugs := make([]client.ProvisionUserGroup, len(ugIDs))
			for j, id := range ugIDs {
				ugs[j] = client.ProvisionUserGroup{ID: id}
			}
			groups[i] = client.ProvisionGroup{
				Name:       gm.Name.ValueString(),
				RoleID:     gm.RoleID.ValueString(),
				UserGroups: ugs,
			}
		}
		ud.ProvisionGroups = groups
	} else {
		ud.ProvisionGroups = []client.ProvisionGroup{}
	}

	if !m.ProvisionMedia.IsNull() && !m.ProvisionMedia.IsUnknown() {
		var mediaModels []ProvisionMediaModel
		diags.Append(m.ProvisionMedia.ElementsAs(ctx, &mediaModels, false)...)
		if diags.HasError() {
			return ud, diags
		}
		media := make([]client.ProvisionMedia, len(mediaModels))
		for i, mm := range mediaModels {
			media[i] = client.ProvisionMedia{
				Name:        mm.Name.ValueString(),
				MediaTypeID: mm.MediaTypeID.ValueString(),
				Attribute:   mm.Attribute.ValueString(),
				Active:      provisionMediaActiveMap[mm.Active.ValueString()],
				Severity:    mm.Severity.ValueInt64(),
				Period:      mm.Period.ValueString(),
			}
		}
		ud.ProvisionMedia = media
	} else {
		ud.ProvisionMedia = []client.ProvisionMedia{}
	}

	return ud, diags
}

func userDirectoryBaseToModel(ctx context.Context, ud *client.UserDirectory, m *UserDirectoryBaseModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Name = types.StringValue(ud.Name)
	m.Description = types.StringValue(ud.Description)
	m.ProvisionStatus = types.StringValue(provisionStatusReverseMap[ud.ProvisionStatus])
	m.GroupName = types.StringValue(ud.GroupName)
	m.UserUsername = types.StringValue(ud.UserUsername)
	m.UserLastname = types.StringValue(ud.UserLastname)

	groupModels := make([]ProvisionGroupModel, len(ud.ProvisionGroups))
	for i, g := range ud.ProvisionGroups {
		ugVals := make([]attr.Value, len(g.UserGroups))
		for j, ug := range g.UserGroups {
			ugVals[j] = types.StringValue(ug.ID)
		}
		ugSet, d := types.SetValue(types.StringType, ugVals)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		groupModels[i] = ProvisionGroupModel{
			Name:       types.StringValue(g.Name),
			RoleID:     types.StringValue(g.RoleID),
			UserGroups: ugSet,
		}
	}
	var d diag.Diagnostics
	m.ProvisionGroups, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: provisionGroupAttrTypes}, groupModels)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	mediaModels := make([]ProvisionMediaModel, len(ud.ProvisionMedia))
	for i, pm := range ud.ProvisionMedia {
		mediaModels[i] = ProvisionMediaModel{
			Name:        types.StringValue(pm.Name),
			MediaTypeID: types.StringValue(pm.MediaTypeID),
			Attribute:   types.StringValue(pm.Attribute),
			Active:      types.StringValue(provisionMediaActiveReverseMap[pm.Active]),
			Severity:    types.Int64Value(pm.Severity),
			Period:      types.StringValue(pm.Period),
		}
	}
	m.ProvisionMedia, d = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: provisionMediaAttrTypes}, mediaModels)
	diags.Append(d...)
	return diags
}

func lookupUserDirectory(ctx context.Context, c client.Client, idpType client.IDPType, id, name types.String) (*client.UserDirectory, diag.Diagnostics) {
	var diags diag.Diagnostics

	if id.IsNull() && name.IsNull() {
		diags.AddError("Missing lookup key", "Exactly one of `id` or `name` must be set.")
		return nil, diags
	}

	typeName := "LDAP"
	if idpType == client.IDPTypeSAML {
		typeName = "SAML"
	}

	if !id.IsNull() {
		ud, err := client.UserDirectoryGet(ctx, c, id.ValueString())
		if err != nil {
			diags.AddError("Error reading user directory", err.Error())
			return nil, diags
		}
		if ud == nil {
			diags.AddError("User directory not found", fmt.Sprintf("No user directory found with id %q.", id.ValueString()))
			return nil, diags
		}
		if ud.IDPType != idpType {
			diags.AddError("Wrong user directory type", fmt.Sprintf("User directory %q is not a %s directory.", id.ValueString(), typeName))
			return nil, diags
		}
		return ud, diags
	}

	dirs, err := client.UserDirectoryGetByName(ctx, c, name.ValueString(), idpType)
	if err != nil {
		diags.AddError("Error reading user directory", err.Error())
		return nil, diags
	}
	switch len(dirs) {
	case 0:
		diags.AddError("User directory not found", fmt.Sprintf("No %s user directory found with name %q.", typeName, name.ValueString()))
	case 1:
		return &dirs[0], diags
	default:
		diags.AddError("Multiple user directories found", fmt.Sprintf("Found %d %s user directories with name %q; use `id` to disambiguate.", len(dirs), typeName, name.ValueString()))
	}
	return nil, diags
}
