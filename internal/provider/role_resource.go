package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}
var _ resource.ResourceWithModifyPlan = &RoleResource{}

var (
	roleTypeMap = map[string]int64{
		"user": 1, "admin": 2, "super_admin": 3,
	}
	roleTypeReverseMap = map[int64]string{
		1: "user", 2: "admin", 3: "super_admin",
	}

	uiItemAttrTypes = map[string]attr.Type{
		"name":    types.StringType,
		"enabled": types.BoolType,
	}
	moduleItemAttrTypes = map[string]attr.Type{
		"module_id": types.StringType,
		"enabled":   types.BoolType,
	}
	actionItemAttrTypes = map[string]attr.Type{
		"name":    types.StringType,
		"enabled": types.BoolType,
	}
	rulesAttrTypes = map[string]attr.Type{
		"ui":                     types.ListType{ElemType: types.ObjectType{AttrTypes: uiItemAttrTypes}},
		"ui_default_access":      types.BoolType,
		"modules_default_access": types.BoolType,
		"actions_default_access": types.BoolType,
		"api_access":             types.BoolType,
		"api_mode":               types.StringType,
		"api_methods":            types.SetType{ElemType: types.StringType},
		"modules":                types.ListType{ElemType: types.ObjectType{AttrTypes: moduleItemAttrTypes}},
		"actions":                types.ListType{ElemType: types.ObjectType{AttrTypes: actionItemAttrTypes}},
	}
)

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

type RoleResource struct {
	client client.Client
}

type RoleResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Type  types.String `tfsdk:"type"`
	Rules types.Object `tfsdk:"rules"`
}

// RoleRulesModel is used for marshaling/unmarshaling types.Object ↔ struct.
type RoleRulesModel struct {
	UI                   types.List   `tfsdk:"ui"`
	UIDefaultAccess      types.Bool   `tfsdk:"ui_default_access"`
	ModulesDefaultAccess types.Bool   `tfsdk:"modules_default_access"`
	ActionsDefaultAccess types.Bool   `tfsdk:"actions_default_access"`
	APIAccess            types.Bool   `tfsdk:"api_access"`
	APIMode              types.String `tfsdk:"api_mode"`
	APIMethods           types.Set    `tfsdk:"api_methods"`
	Modules              types.List   `tfsdk:"modules"`
	Actions              types.List   `tfsdk:"actions"`
}

type RuleUIModel struct {
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

type RuleModuleModel struct {
	ModuleID types.String `tfsdk:"module_id"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

type RuleActionModel struct {
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the role. Must be unique within Zabbix.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Role type. One of: `user`, `admin`, `super_admin`.",
				Validators: []validator.String{
					stringvalidator.OneOf("user", "admin", "super_admin"),
				},
			},
			"rules": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Access rules for the role.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"ui": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "UI element access rules.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Name of the UI element.",
								},
								"enabled": schema.BoolAttribute{
									Optional:            true,
									Computed:            true,
									Default:             booldefault.StaticBool(true),
									MarkdownDescription: "Whether access to this UI element is enabled.",
								},
							},
						},
					},
					"ui_default_access": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Default access for UI elements not listed in `ui`. Defaults to `true`.",
					},
					"modules_default_access": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Default access for modules not listed in `modules`. Defaults to `true`.",
					},
					"actions_default_access": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Default access for actions not listed in `actions`. Defaults to `true`.",
					},
					"api_access": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Whether API access is enabled for this role. Defaults to `true`.",
					},
					"api_mode": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "API access mode. One of: `deny`, `allow`. Defaults to `deny`.",
						Validators: []validator.String{
							stringvalidator.OneOf("deny", "allow"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"api_methods": schema.SetAttribute{
						Optional:            true,
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "API methods affected by `api_mode`.",
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"modules": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Module access rules.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"module_id": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Module ID.",
								},
								"enabled": schema.BoolAttribute{
									Optional:            true,
									Computed:            true,
									Default:             booldefault.StaticBool(true),
									MarkdownDescription: "Whether access to this module is enabled.",
								},
							},
						},
					},
					"actions": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Action access rules.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Name of the action.",
								},
								"enabled": schema.BoolAttribute{
									Optional:            true,
									Computed:            true,
									Default:             booldefault.StaticBool(true),
									MarkdownDescription: "Whether this action is enabled.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := client.Role{
		Name: data.Name.ValueString(),
		Type: roleTypeMap[data.Type.ValueString()],
	}
	if !data.Rules.IsNull() && !data.Rules.IsUnknown() {
		rules, diags := objectToClientRules(ctx, data.Rules)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		role.Rules = rules
		role.HasRules = true
	}

	id, err := client.RoleCreate(ctx, r.client, role)
	if err != nil {
		resp.Diagnostics.AddError("Error creating role", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.RoleGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading role after create", err.Error())
		return
	}
	if created != nil {
		rulesObj, d := clientRulesToObject(created.Rules)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Rules = rulesObj
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := client.RoleGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.Name = types.StringValue(role.Name)
	data.Type = types.StringValue(roleTypeReverseMap[role.Type])
	rulesObj, diags := clientRulesToObject(role.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Rules = rulesObj
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := client.Role{
		ID:   data.ID.ValueString(),
		Name: data.Name.ValueString(),
		Type: roleTypeMap[data.Type.ValueString()],
	}
	if !data.Rules.IsNull() && !data.Rules.IsUnknown() {
		rules, diags := objectToClientRules(ctx, data.Rules)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		role.Rules = rules
		role.HasRules = true
	}

	if err := client.RoleUpdate(ctx, r.client, role); err != nil {
		resp.Diagnostics.AddError("Error updating role", err.Error())
		return
	}

	updated, err := client.RoleGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading role after update", err.Error())
		return
	}
	if updated != nil {
		rulesObj, d := clientRulesToObject(updated.Rules)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Rules = rulesObj
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.RoleDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting role", err.Error())
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan marks rules as unknown when the role type is changing and rules are not in config.
// Zabbix's default UI rules differ between role types, so we can't predict the post-update set.
func (r *RoleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy (plan is null) or create (state is null): nothing to stabilize.
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var configRules types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("rules"), &configRules)...)
	if resp.Diagnostics.HasError() || !configRules.IsNull() {
		return
	}
	var planType, stateType types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("type"), &planType)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("type"), &stateType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !planType.Equal(stateType) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("rules"), types.ObjectUnknown(rulesAttrTypes))...)
	}
}

// clientRulesToObject converts client.RoleRules to a types.Object for the framework.
func clientRulesToObject(r client.RoleRules) (types.Object, diag.Diagnostics) {
	uiItems := make([]attr.Value, len(r.UI))
	for i, u := range r.UI {
		obj, d := types.ObjectValue(uiItemAttrTypes, map[string]attr.Value{
			"name":    types.StringValue(u.Name),
			"enabled": types.BoolValue(statusToBool(u.Status)),
		})
		if d.HasError() {
			return types.ObjectNull(rulesAttrTypes), d
		}
		uiItems[i] = obj
	}
	uiList, d := types.ListValue(types.ObjectType{AttrTypes: uiItemAttrTypes}, uiItems)
	if d.HasError() {
		return types.ObjectNull(rulesAttrTypes), d
	}

	moduleItems := make([]attr.Value, len(r.Modules))
	for i, m := range r.Modules {
		obj, d := types.ObjectValue(moduleItemAttrTypes, map[string]attr.Value{
			"module_id": types.StringValue(strconv.FormatInt(m.ModuleID, 10)),
			"enabled":   types.BoolValue(statusToBool(m.Status)),
		})
		if d.HasError() {
			return types.ObjectNull(rulesAttrTypes), d
		}
		moduleItems[i] = obj
	}
	moduleList, d := types.ListValue(types.ObjectType{AttrTypes: moduleItemAttrTypes}, moduleItems)
	if d.HasError() {
		return types.ObjectNull(rulesAttrTypes), d
	}

	actionItems := make([]attr.Value, len(r.Actions))
	for i, a := range r.Actions {
		obj, d := types.ObjectValue(actionItemAttrTypes, map[string]attr.Value{
			"name":    types.StringValue(a.Name),
			"enabled": types.BoolValue(statusToBool(a.Status)),
		})
		if d.HasError() {
			return types.ObjectNull(rulesAttrTypes), d
		}
		actionItems[i] = obj
	}
	actionList, d := types.ListValue(types.ObjectType{AttrTypes: actionItemAttrTypes}, actionItems)
	if d.HasError() {
		return types.ObjectNull(rulesAttrTypes), d
	}

	methods := make([]attr.Value, len(r.APIMethods))
	for i, m := range r.APIMethods {
		methods[i] = types.StringValue(m)
	}
	methodSet, d := types.SetValue(types.StringType, methods)
	if d.HasError() {
		return types.ObjectNull(rulesAttrTypes), d
	}

	return types.ObjectValue(rulesAttrTypes, map[string]attr.Value{
		"ui":                     uiList,
		"ui_default_access":      types.BoolValue(flagToBool(r.UIDefaultAccess)),
		"modules_default_access": types.BoolValue(flagToBool(r.ModulesDefaultAccess)),
		"actions_default_access": types.BoolValue(flagToBool(r.ActionsDefaultAccess)),
		"api_access":             types.BoolValue(flagToBool(r.APIAccess)),
		"api_mode":               types.StringValue(intToAPIMode(r.APIMode)),
		"api_methods":            methodSet,
		"modules":                moduleList,
		"actions":                actionList,
	})
}

// objectToClientRules converts a types.Object to client.RoleRules.
func objectToClientRules(ctx context.Context, obj types.Object) (client.RoleRules, diag.Diagnostics) {
	var m RoleRulesModel
	if d := obj.As(ctx, &m, basetypes.ObjectAsOptions{}); d.HasError() {
		return client.RoleRules{}, d
	}

	var uiModels []RuleUIModel
	if d := m.UI.ElementsAs(ctx, &uiModels, false); d.HasError() {
		return client.RoleRules{}, d
	}
	ui := make([]client.RuleUI, len(uiModels))
	for i, u := range uiModels {
		ui[i] = client.RuleUI{Name: u.Name.ValueString(), Status: boolToStatus(u.Enabled.ValueBool())}
	}

	var moduleModels []RuleModuleModel
	if d := m.Modules.ElementsAs(ctx, &moduleModels, false); d.HasError() {
		return client.RoleRules{}, d
	}
	modules := make([]client.RuleModule, len(moduleModels))
	for i, mod := range moduleModels {
		mid, _ := strconv.ParseInt(mod.ModuleID.ValueString(), 10, 64)
		modules[i] = client.RuleModule{ModuleID: mid, Status: boolToStatus(mod.Enabled.ValueBool())}
	}

	var actionModels []RuleActionModel
	if d := m.Actions.ElementsAs(ctx, &actionModels, false); d.HasError() {
		return client.RoleRules{}, d
	}
	actions := make([]client.RuleAction, len(actionModels))
	for i, a := range actionModels {
		actions[i] = client.RuleAction{Name: a.Name.ValueString(), Status: boolToStatus(a.Enabled.ValueBool())}
	}

	var methodStrings []string
	if d := m.APIMethods.ElementsAs(ctx, &methodStrings, false); d.HasError() {
		return client.RoleRules{}, d
	}

	return client.RoleRules{
		UI:                   ui,
		UIDefaultAccess:      boolToFlag(m.UIDefaultAccess.ValueBool()),
		ModulesDefaultAccess: boolToFlag(m.ModulesDefaultAccess.ValueBool()),
		ActionsDefaultAccess: boolToFlag(m.ActionsDefaultAccess.ValueBool()),
		APIAccess:            boolToFlag(m.APIAccess.ValueBool()),
		APIMode:              apiModeToInt(m.APIMode.ValueString()),
		APIMethods:           methodStrings,
		Modules:              modules,
		Actions:              actions,
	}, nil
}

// In Zabbix, rule status 0=enabled, 1=disabled.
func boolToStatus(enabled bool) int64 {
	if enabled {
		return 0
	}
	return 1
}

func statusToBool(status int64) bool {
	return status == 0
}

// In Zabbix, flag 1=enabled, 0=disabled (ui.default_access, api.access, etc.).
func boolToFlag(enabled bool) int64 {
	if enabled {
		return 1
	}
	return 0
}

func flagToBool(flag int64) bool {
	return flag == 1
}

// api.mode: 0=deny listed methods, 1=allow only listed methods.
func apiModeToInt(mode string) int64 {
	if mode == "allow" {
		return 1
	}
	return 0
}

func intToAPIMode(mode int64) string {
	if mode == 1 {
		return "allow"
	}
	return "deny"
}
