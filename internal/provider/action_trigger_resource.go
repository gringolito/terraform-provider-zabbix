package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &ActionTriggerResource{}
var _ resource.ResourceWithImportState = &ActionTriggerResource{}
var _ resource.ResourceWithConfigValidators = &ActionTriggerResource{}

// --- Enum maps ---

var (
	actionStatusMap = map[string]int64{
		"enabled": 0, "disabled": 1,
	}
	actionStatusReverseMap = map[int64]string{
		0: "enabled", 1: "disabled",
	}
	actionEvalTypeMap = map[string]int64{
		"and_or": 0, "and": 1, "or": 2, "custom_expression": 3,
	}
	actionEvalTypeReverseMap = map[int64]string{
		0: "and_or", 1: "and", 2: "or", 3: "custom_expression",
	}
	actionConditionTypeMap = map[string]int64{
		"host_group":         0,
		"host":               1,
		"trigger":            2,
		"trigger_name":       3,
		"trigger_severity":   4,
		"time_period":        5,
		"host_ip":            6,
		"maintenance_status": 16,
		"event_tag":          25,
		"event_tag_value":    26,
	}
	actionConditionTypeReverseMap = map[int64]string{
		0:  "host_group",
		1:  "host",
		2:  "trigger",
		3:  "trigger_name",
		4:  "trigger_severity",
		5:  "time_period",
		6:  "host_ip",
		16: "maintenance_status",
		25: "event_tag",
		26: "event_tag_value",
	}
	actionOperatorMap = map[string]int64{
		"equals":            0,
		"not_equals":        1,
		"like":              2,
		"not_like":          3,
		"in":                4,
		"greater_or_equals": 5,
		"less_or_equals":    6,
		"not_in":            7,
	}
	actionOperatorReverseMap = map[int64]string{
		0: "equals",
		1: "not_equals",
		2: "like",
		3: "not_like",
		4: "in",
		5: "greater_or_equals",
		6: "less_or_equals",
		7: "not_in",
	}
	actionExecuteOnMap = map[string]int64{
		"agent": 0, "server_or_proxy": 1, "server": 2,
	}
	actionExecuteOnReverseMap = map[int64]string{
		0: "agent", 1: "server_or_proxy", 2: "server",
	}
	actionSSHAuthTypeMap = map[string]int64{
		"password": 0, "public_key": 1,
	}
	actionSSHAuthTypeReverseMap = map[int64]string{
		0: "password", 1: "public_key",
	}
)

// --- Attr type maps for nested objects ---

var actionConditionAttrTypes = map[string]attr.Type{
	"condition_type": types.StringType,
	"operator":       types.StringType,
	"value":          types.StringType,
	"value2":         types.StringType,
	"label":          types.StringType,
}

var actionFilterAttrTypes = map[string]attr.Type{
	"evaluation_type": types.StringType,
	"formula":         types.StringType,
	"condition":       types.ListType{ElemType: types.ObjectType{AttrTypes: actionConditionAttrTypes}},
}

var actionSendMessageAttrTypes = map[string]attr.Type{
	"use_default_message": types.BoolType,
	"subject":             types.StringType,
	"message":             types.StringType,
	"media_type_id":       types.StringType,
	"user_group_ids":      types.SetType{ElemType: types.StringType},
	"user_ids":            types.SetType{ElemType: types.StringType},
}

var actionCustomScriptAttrTypes = map[string]attr.Type{
	"command":    types.StringType,
	"execute_on": types.StringType,
}

var actionIPMIAttrTypes = map[string]attr.Type{
	"command": types.StringType,
}

var actionSSHAttrTypes = map[string]attr.Type{
	"command":     types.StringType,
	"authtype":    types.StringType,
	"username":    types.StringType,
	"password":    types.StringType,
	"public_key":  types.StringType,
	"private_key": types.StringType,
	"port":        types.StringType,
}

var actionTelnetAttrTypes = map[string]attr.Type{
	"command":  types.StringType,
	"username": types.StringType,
	"password": types.StringType,
	"port":     types.StringType,
}

var actionGlobalScriptAttrTypes = map[string]attr.Type{
	"script_id": types.StringType,
}

var actionRemoteCommandAttrTypes = map[string]attr.Type{
	"current_host":   types.BoolType,
	"host_ids":       types.SetType{ElemType: types.StringType},
	"host_group_ids": types.SetType{ElemType: types.StringType},
	"custom_script":  types.ObjectType{AttrTypes: actionCustomScriptAttrTypes},
	"ipmi":           types.ObjectType{AttrTypes: actionIPMIAttrTypes},
	"ssh":            types.ObjectType{AttrTypes: actionSSHAttrTypes},
	"telnet":         types.ObjectType{AttrTypes: actionTelnetAttrTypes},
	"global_script":  types.ObjectType{AttrTypes: actionGlobalScriptAttrTypes},
}

var actionOperationAttrTypes = map[string]attr.Type{
	"escalation_period":    types.StringType,
	"escalation_step_from": types.Int64Type,
	"escalation_step_to":   types.Int64Type,
	"send_message":         types.ObjectType{AttrTypes: actionSendMessageAttrTypes},
	"remote_command":       types.ObjectType{AttrTypes: actionRemoteCommandAttrTypes},
}

var actionRecoveryOpAttrTypes = map[string]attr.Type{
	"notify_all_involved": types.BoolType,
	"send_message":        types.ObjectType{AttrTypes: actionSendMessageAttrTypes},
	"remote_command":      types.ObjectType{AttrTypes: actionRemoteCommandAttrTypes},
}

// --- Model types ---

type ActionTriggerResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Status             types.String `tfsdk:"status"`
	EscalationPeriod   types.String `tfsdk:"escalation_period"`
	PauseSuppressed    types.Bool   `tfsdk:"pause_suppressed"`
	NotifyIfCanceled   types.Bool   `tfsdk:"notify_if_canceled"`
	Filter             types.Object `tfsdk:"filter"`
	Operations         types.List   `tfsdk:"operations"`
	RecoveryOperations types.List   `tfsdk:"recovery_operations"`
	UpdateOperations   types.List   `tfsdk:"update_operations"`
}

type ActionFilterModel struct {
	EvaluationType types.String `tfsdk:"evaluation_type"`
	Formula        types.String `tfsdk:"formula"`
	Conditions     types.List   `tfsdk:"condition"`
}

type ActionConditionModel struct {
	ConditionType types.String `tfsdk:"condition_type"`
	Operator      types.String `tfsdk:"operator"`
	Value         types.String `tfsdk:"value"`
	Value2        types.String `tfsdk:"value2"`
	Label         types.String `tfsdk:"label"`
}

type ActionOperationModel struct {
	EscalationPeriod   types.String `tfsdk:"escalation_period"`
	EscalationStepFrom types.Int64  `tfsdk:"escalation_step_from"`
	EscalationStepTo   types.Int64  `tfsdk:"escalation_step_to"`
	SendMessage        types.Object `tfsdk:"send_message"`
	RemoteCommand      types.Object `tfsdk:"remote_command"`
}

type ActionRecoveryOpModel struct {
	NotifyAllInvolved types.Bool   `tfsdk:"notify_all_involved"`
	SendMessage       types.Object `tfsdk:"send_message"`
	RemoteCommand     types.Object `tfsdk:"remote_command"`
}

type ActionSendMessageModel struct {
	UseDefaultMessage types.Bool   `tfsdk:"use_default_message"`
	Subject           types.String `tfsdk:"subject"`
	Message           types.String `tfsdk:"message"`
	MediaTypeID       types.String `tfsdk:"media_type_id"`
	UserGroupIDs      types.Set    `tfsdk:"user_group_ids"`
	UserIDs           types.Set    `tfsdk:"user_ids"`
}

type ActionRemoteCommandModel struct {
	CurrentHost  types.Bool   `tfsdk:"current_host"`
	HostIDs      types.Set    `tfsdk:"host_ids"`
	HostGroupIDs types.Set    `tfsdk:"host_group_ids"`
	CustomScript types.Object `tfsdk:"custom_script"`
	IPMI         types.Object `tfsdk:"ipmi"`
	SSH          types.Object `tfsdk:"ssh"`
	Telnet       types.Object `tfsdk:"telnet"`
	GlobalScript types.Object `tfsdk:"global_script"`
}

type ActionCustomScriptModel struct {
	Command   types.String `tfsdk:"command"`
	ExecuteOn types.String `tfsdk:"execute_on"`
}

type ActionIPMIModel struct {
	Command types.String `tfsdk:"command"`
}

type ActionSSHModel struct {
	Command    types.String `tfsdk:"command"`
	AuthType   types.String `tfsdk:"authtype"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
	Port       types.String `tfsdk:"port"`
}

type ActionTelnetModel struct {
	Command  types.String `tfsdk:"command"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Port     types.String `tfsdk:"port"`
}

type ActionGlobalScriptModel struct {
	ScriptID types.String `tfsdk:"script_id"`
}

// --- Resource ---

func NewActionTriggerResource() resource.Resource {
	return &ActionTriggerResource{}
}

type ActionTriggerResource struct {
	client client.Client
}

func (r *ActionTriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_trigger"
}

func (r *ActionTriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	sendMessageAttrs := map[string]schema.Attribute{
		"use_default_message": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
			MarkdownDescription: "When `true`, the media type's default subject/message are used; `subject` and `message` must be absent. When `false`, `subject` and `message` are required.",
		},
		"subject": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Custom message subject. Required when `use_default_message = false`, must be absent when `true`.",
		},
		"message": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			MarkdownDescription: "Custom message body. Required when `use_default_message = false`, must be absent when `true`.",
		},
		"media_type_id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("0"),
			MarkdownDescription: "ID of the media type to use. Defaults to `\"0\"` (all media types configured for the recipient).",
		},
		"user_group_ids": schema.SetAttribute{
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Set of user group IDs to send the message to.",
		},
		"user_ids": schema.SetAttribute{
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Set of user IDs to send the message to.",
		},
	}

	sendMessageBlock := schema.SingleNestedBlock{
		MarkdownDescription: "Send a notification message.",
		Attributes:          sendMessageAttrs,
	}

	remoteCommandBlock := schema.SingleNestedBlock{
		MarkdownDescription: "Execute a remote command.",
		Attributes: map[string]schema.Attribute{
			"current_host": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Run the command on the host that triggered the event. Defaults to `false`.",
			},
			"host_ids": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of host IDs to run the command on.",
			},
			"host_group_ids": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of host group IDs to run the command on.",
			},
		},
		Blocks: map[string]schema.Block{
			"custom_script": schema.SingleNestedBlock{
				MarkdownDescription: "Execute a custom script. Exactly one of `custom_script`, `ipmi`, `ssh`, `telnet`, `global_script` must be set.",
				Attributes: map[string]schema.Attribute{
					"command": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Script body to execute.",
					},
					"execute_on": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Execution target. One of: `agent`, `server_or_proxy`, `server`.",
						Validators: []validator.String{
							stringvalidator.OneOf("agent", "server_or_proxy", "server"),
						},
					},
				},
			},
			"ipmi": schema.SingleNestedBlock{
				MarkdownDescription: "Execute an IPMI command. Exactly one of `custom_script`, `ipmi`, `ssh`, `telnet`, `global_script` must be set.",
				Attributes: map[string]schema.Attribute{
					"command": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "IPMI command to execute.",
					},
				},
			},
			"ssh": schema.SingleNestedBlock{
				MarkdownDescription: "Execute a command over SSH. Exactly one of `custom_script`, `ipmi`, `ssh`, `telnet`, `global_script` must be set.",
				Attributes: map[string]schema.Attribute{
					"command": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Command to execute over SSH.",
					},
					"authtype": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Authentication type. One of: `password`, `public_key`.",
						Validators: []validator.String{
							stringvalidator.OneOf("password", "public_key"),
						},
					},
					"username": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "SSH username.",
					},
					"password": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SSH password (used when `authtype = \"password\"`).",
					},
					"public_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "Public key for SSH authentication (used when `authtype = \"public_key\"`).",
					},
					"private_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "Private key for SSH authentication (used when `authtype = \"public_key\"`).",
					},
					"port": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SSH port. Defaults to `\"\"` (Zabbix uses 22).",
					},
				},
			},
			"telnet": schema.SingleNestedBlock{
				MarkdownDescription: "Execute a command over Telnet. Exactly one of `custom_script`, `ipmi`, `ssh`, `telnet`, `global_script` must be set.",
				Attributes: map[string]schema.Attribute{
					"command": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Command to execute over Telnet.",
					},
					"username": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Telnet username.",
					},
					"password": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Telnet password.",
					},
					"port": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "Telnet port. Defaults to `\"\"` (Zabbix uses 23).",
					},
				},
			},
			"global_script": schema.SingleNestedBlock{
				MarkdownDescription: "Run a Zabbix global script. Exactly one of `custom_script`, `ipmi`, `ssh`, `telnet`, `global_script` must be set.",
				Attributes: map[string]schema.Attribute{
					"script_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "ID of the Zabbix global script to run.",
					},
				},
			},
		},
	}

	opBlocks := map[string]schema.Block{
		"send_message":   sendMessageBlock,
		"remote_command": remoteCommandBlock,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix trigger action (`event_source = 0`).\n\n" +
			"Trigger actions fire when a trigger changes state (problem/recovery/update). " +
			"The `event_source` discriminator is hardcoded to `0` — see [ADR-0015](../docs/adr/0015-action-trigger-as-typed-resource.md).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the trigger action.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the trigger action. Must be unique within Zabbix.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Whether the action is active. One of: `enabled`, `disabled`. Defaults to `enabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"escalation_period": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1h"),
				MarkdownDescription: "Default escalation period. Accepts time suffixes (`\"1h\"`) and macros (`\"{$ESC_PERIOD}\"`). Defaults to `\"1h\"`.",
			},
			"pause_suppressed": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Pause operations while hosts are in a maintenance period. Defaults to `true`.",
			},
			"notify_if_canceled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Send recovery notifications for problems canceled during maintenance. Defaults to `true`.",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Conditions that must be matched for the action to fire.",
				Attributes: map[string]schema.Attribute{
					"evaluation_type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "How conditions are evaluated. One of: `and_or`, `and`, `or`, `custom_expression`.",
						Validators: []validator.String{
							stringvalidator.OneOf("and_or", "and", "or", "custom_expression"),
						},
					},
					"formula": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "Custom expression joining condition labels (e.g. `\"{A} and ({B} or {C})\"`). Required when `evaluation_type = \"custom_expression\"`, must be absent otherwise.",
					},
				},
				Blocks: map[string]schema.Block{
					"condition": schema.ListNestedBlock{
						MarkdownDescription: "Filter conditions. At least one condition is required.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"condition_type": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Type of condition. One of: `host_group`, `host`, `trigger`, `trigger_name`, `trigger_severity`, `time_period`, `host_ip`, `maintenance_status`, `event_tag`, `event_tag_value`.",
									Validators: []validator.String{
										stringvalidator.OneOf(
											"host_group", "host", "trigger", "trigger_name",
											"trigger_severity", "time_period", "host_ip",
											"maintenance_status", "event_tag", "event_tag_value",
										),
									},
								},
								"operator": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Comparison operator. One of: `equals`, `not_equals`, `like`, `not_like`, `in`, `greater_or_equals`, `less_or_equals`, `not_in`.",
									Validators: []validator.String{
										stringvalidator.OneOf(
											"equals", "not_equals", "like", "not_like",
											"in", "greater_or_equals", "less_or_equals", "not_in",
										),
									},
								},
								"value": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Value to compare against.",
								},
								"value2": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString(""),
									MarkdownDescription: "Secondary comparison value (used by some condition types).",
								},
								"label": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString(""),
									MarkdownDescription: "Single-letter label referenced in `formula` (e.g. `\"A\"`). Required when `evaluation_type = \"custom_expression\"`, must be absent otherwise.",
								},
							},
						},
					},
				},
			},
			"operations": schema.ListNestedBlock{
				MarkdownDescription: "Escalation operation steps. Each step runs `send_message` or `remote_command`.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"escalation_period": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("0"),
							MarkdownDescription: "Escalation period for this step. `\"0\"` inherits the action's `escalation_period`. Accepts time suffixes (`\"1h\"`) and macros.",
						},
						"escalation_step_from": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1),
							MarkdownDescription: "First escalation step this operation applies to. Defaults to `1`.",
						},
						"escalation_step_to": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1),
							MarkdownDescription: "Last escalation step this operation applies to. `0` means all steps from `escalation_step_from` onwards. Defaults to `1`.",
						},
					},
					Blocks: opBlocks,
				},
			},
			"recovery_operations": schema.ListNestedBlock{
				MarkdownDescription: "Operations executed when the triggering problem is resolved.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"notify_all_involved": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Notify all users who previously received problem notifications. Mutually exclusive with `send_message` and `remote_command`.",
						},
					},
					Blocks: opBlocks,
				},
			},
			"update_operations": schema.ListNestedBlock{
				MarkdownDescription: "Operations executed when the triggering problem is updated (acknowledged).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"notify_all_involved": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Notify all users who previously received problem notifications. Mutually exclusive with `send_message` and `remote_command`.",
						},
					},
					Blocks: opBlocks,
				},
			},
		},
	}
}

// --- ConfigValidators ---

func (r *ActionTriggerResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		actionFilterExpressionValidator{},
		actionSendMessageDefaultValidator{},
	}
}

// actionFilterExpressionValidator enforces:
//   - filter.formula must be set iff evaluation_type = "custom_expression"
//   - each condition.label must be set iff evaluation_type = "custom_expression"
type actionFilterExpressionValidator struct{}

func (v actionFilterExpressionValidator) Description(_ context.Context) string {
	return `"formula" and condition "label" must be set iff "evaluation_type" = "custom_expression".`
}

func (v actionFilterExpressionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v actionFilterExpressionValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ActionTriggerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Filter.IsUnknown() {
		return
	}
	if data.Filter.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("filter"),
			"Missing required block",
			`A "filter" block is required.`,
		)
		return
	}

	var filterModel ActionFilterModel
	resp.Diagnostics.Append(data.Filter.As(ctx, &filterModel, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if filterModel.EvaluationType.IsUnknown() {
		return
	}

	evalType := filterModel.EvaluationType.ValueString()
	formula := filterModel.Formula.ValueString()

	if evalType == "custom_expression" && formula == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("filter").AtName("formula"),
			"Missing formula",
			`"formula" must be set when "evaluation_type" = "custom_expression".`,
		)
	}
	if evalType != "custom_expression" && formula != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("filter").AtName("formula"),
			"Unexpected formula",
			fmt.Sprintf(`"formula" must be empty when "evaluation_type" = %q.`, evalType),
		)
	}

	if filterModel.Conditions.IsUnknown() {
		return
	}

	var conditions []ActionConditionModel
	resp.Diagnostics.Append(filterModel.Conditions.ElementsAs(ctx, &conditions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, cond := range conditions {
		if cond.Label.IsUnknown() {
			continue
		}
		label := cond.Label.ValueString()
		condPath := path.Root("filter").AtName("condition").AtListIndex(i).AtName("label")

		if evalType == "custom_expression" && label == "" {
			resp.Diagnostics.AddAttributeError(
				condPath,
				"Missing label",
				fmt.Sprintf(`condition[%d]: "label" must be set when "evaluation_type" = "custom_expression".`, i),
			)
		}
		if evalType != "custom_expression" && label != "" {
			resp.Diagnostics.AddAttributeError(
				condPath,
				"Unexpected label",
				fmt.Sprintf(`condition[%d]: "label" must be empty when "evaluation_type" = %q.`, i, evalType),
			)
		}
	}
}

// actionSendMessageDefaultValidator enforces the use_default_message ↔ subject/message invariant
// across operations, recovery_operations, and update_operations.
type actionSendMessageDefaultValidator struct{}

func (v actionSendMessageDefaultValidator) Description(_ context.Context) string {
	return `send_message: "subject"/"message" must be set iff "use_default_message" = false.`
}

func (v actionSendMessageDefaultValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v actionSendMessageDefaultValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ActionTriggerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateSendMessageInOpList(ctx, resp, path.Root("operations"), data.Operations, actionOperationAttrTypes, false)
	validateSendMessageInOpList(ctx, resp, path.Root("recovery_operations"), data.RecoveryOperations, actionRecoveryOpAttrTypes, false)
	validateSendMessageInOpList(ctx, resp, path.Root("update_operations"), data.UpdateOperations, actionRecoveryOpAttrTypes, false)
}

// validateSendMessageInOpList validates send_message blocks inside a list of operations.
// opAttrTypes must contain "send_message".
func validateSendMessageInOpList(ctx context.Context, resp *resource.ValidateConfigResponse, basePath path.Path, list types.List, _ map[string]attr.Type, _ bool) {
	if list.IsUnknown() || list.IsNull() {
		return
	}

	elems := list.Elements()
	for i, elem := range elems {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsUnknown() || obj.IsNull() {
			continue
		}
		attrs := obj.Attributes()
		smVal, ok := attrs["send_message"]
		if !ok {
			continue
		}
		smObj, ok := smVal.(types.Object)
		if !ok || smObj.IsUnknown() || smObj.IsNull() {
			continue
		}

		var sm ActionSendMessageModel
		diags := smObj.As(ctx, &sm, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			continue
		}

		if sm.UseDefaultMessage.IsUnknown() {
			continue
		}

		useDefault := sm.UseDefaultMessage.ValueBool()
		subject := sm.Subject.ValueString()
		message := sm.Message.ValueString()
		smPath := basePath.AtListIndex(i).AtName("send_message")

		if useDefault && subject != "" {
			resp.Diagnostics.AddAttributeError(
				smPath.AtName("subject"),
				"Unexpected subject",
				`"subject" must be empty when "use_default_message" = true.`,
			)
		}
		if useDefault && message != "" {
			resp.Diagnostics.AddAttributeError(
				smPath.AtName("message"),
				"Unexpected message",
				`"message" must be empty when "use_default_message" = true.`,
			)
		}
		if !useDefault && subject == "" {
			resp.Diagnostics.AddAttributeError(
				smPath.AtName("subject"),
				"Missing subject",
				`"subject" must be set when "use_default_message" = false.`,
			)
		}
		if !useDefault && message == "" {
			resp.Diagnostics.AddAttributeError(
				smPath.AtName("message"),
				"Missing message",
				`"message" must be set when "use_default_message" = false.`,
			)
		}
	}
}

// --- CRUD ---

func (r *ActionTriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ActionTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ActionTriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	a, diags := modelToClientAction(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.ActionCreate(ctx, r.client, a)
	if err != nil {
		resp.Diagnostics.AddError("Error creating trigger action", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.ActionGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger action after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(clientActionToModel(ctx, *created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActionTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ActionTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	a, err := client.ActionGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger action", err.Error())
		return
	}
	if a == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(clientActionToModel(ctx, *a, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActionTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ActionTriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ActionTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	a, diags := modelToClientAction(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.ActionUpdate(ctx, r.client, a); err != nil {
		resp.Diagnostics.AddError("Error updating trigger action", err.Error())
		return
	}

	updated, err := client.ActionGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger action after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(clientActionToModel(ctx, *updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActionTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ActionTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.ActionDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting trigger action", err.Error())
		return
	}
}

func (r *ActionTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Model ↔ Client converters ---

func modelToClientAction(ctx context.Context, data ActionTriggerResourceModel) (client.Action, diag.Diagnostics) {
	var diags diag.Diagnostics

	a := client.Action{
		ActionID:         data.ID.ValueString(),
		Name:             data.Name.ValueString(),
		Status:           actionStatusMap[data.Status.ValueString()],
		EscPeriod:        data.EscalationPeriod.ValueString(),
		PauseSuppressed:  boolToInt64(data.PauseSuppressed.ValueBool()),
		NotifyIfCanceled: boolToInt64(data.NotifyIfCanceled.ValueBool()),
	}

	// filter
	if !data.Filter.IsNull() && !data.Filter.IsUnknown() {
		var fm ActionFilterModel
		diags.Append(data.Filter.As(ctx, &fm, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return a, diags
		}
		f, d := modelToClientFilter(ctx, fm)
		diags.Append(d...)
		if diags.HasError() {
			return a, diags
		}
		a.Filter = f
	}

	// operations
	a.Operations = []client.ActionOperation{}
	if !data.Operations.IsNull() && !data.Operations.IsUnknown() {
		var opModels []ActionOperationModel
		diags.Append(data.Operations.ElementsAs(ctx, &opModels, false)...)
		if !diags.HasError() {
			a.Operations = make([]client.ActionOperation, len(opModels))
			for i, m := range opModels {
				op, d := modelToClientOp(ctx, m)
				diags.Append(d...)
				a.Operations[i] = op
			}
		}
	}

	// recovery_operations
	a.RecoveryOperations = []client.ActionOperation{}
	if !data.RecoveryOperations.IsNull() && !data.RecoveryOperations.IsUnknown() {
		var recs []ActionRecoveryOpModel
		diags.Append(data.RecoveryOperations.ElementsAs(ctx, &recs, false)...)
		if !diags.HasError() {
			a.RecoveryOperations = make([]client.ActionOperation, len(recs))
			for i, m := range recs {
				op, d := modelToClientRecoveryOp(ctx, m)
				diags.Append(d...)
				a.RecoveryOperations[i] = op
			}
		}
	}

	// update_operations
	a.UpdateOperations = []client.ActionOperation{}
	if !data.UpdateOperations.IsNull() && !data.UpdateOperations.IsUnknown() {
		var upds []ActionRecoveryOpModel
		diags.Append(data.UpdateOperations.ElementsAs(ctx, &upds, false)...)
		if !diags.HasError() {
			a.UpdateOperations = make([]client.ActionOperation, len(upds))
			for i, m := range upds {
				op, d := modelToClientRecoveryOp(ctx, m)
				diags.Append(d...)
				a.UpdateOperations[i] = op
			}
		}
	}

	return a, diags
}

func modelToClientFilter(ctx context.Context, fm ActionFilterModel) (client.ActionFilter, diag.Diagnostics) {
	var diags diag.Diagnostics
	f := client.ActionFilter{
		EvalType: actionEvalTypeMap[fm.EvaluationType.ValueString()],
		Formula:  fm.Formula.ValueString(),
	}

	if !fm.Conditions.IsNull() && !fm.Conditions.IsUnknown() {
		var condModels []ActionConditionModel
		diags.Append(fm.Conditions.ElementsAs(ctx, &condModels, false)...)
		if !diags.HasError() {
			f.Conditions = make([]client.ActionCondition, len(condModels))
			for i, cm := range condModels {
				f.Conditions[i] = client.ActionCondition{
					ConditionType: actionConditionTypeMap[cm.ConditionType.ValueString()],
					Operator:      actionOperatorMap[cm.Operator.ValueString()],
					Value:         cm.Value.ValueString(),
					Value2:        cm.Value2.ValueString(),
					FormulaID:     cm.Label.ValueString(),
				}
			}
		}
	} else {
		f.Conditions = []client.ActionCondition{}
	}

	return f, diags
}

func modelToClientOp(ctx context.Context, m ActionOperationModel) (client.ActionOperation, diag.Diagnostics) {
	var diags diag.Diagnostics
	op := client.ActionOperation{
		EscPeriod:   m.EscalationPeriod.ValueString(),
		EscStepFrom: m.EscalationStepFrom.ValueInt64(),
		EscStepTo:   m.EscalationStepTo.ValueInt64(),
	}
	d := modelToClientOpMessage(ctx, m.SendMessage, m.RemoteCommand, &op)
	diags.Append(d...)
	return op, diags
}

func modelToClientRecoveryOp(ctx context.Context, m ActionRecoveryOpModel) (client.ActionOperation, diag.Diagnostics) {
	var diags diag.Diagnostics
	op := client.ActionOperation{}

	if m.NotifyAllInvolved.ValueBool() {
		op.OperationType = 11
		// Zabbix 7.0 requires opmessage for type 11 but rejects mediatypeid,
		// opmessage_grp and opmessage_usr.
		op.OpMessage = &client.ActionOpMessage{UseDefault: 1}
		return op, diags
	}

	d := modelToClientOpMessage(ctx, m.SendMessage, m.RemoteCommand, &op)
	diags.Append(d...)
	return op, diags
}

func modelToClientOpMessage(ctx context.Context, sendMsg types.Object, remoteCmd types.Object, op *client.ActionOperation) diag.Diagnostics {
	var diags diag.Diagnostics

	if !sendMsg.IsNull() && !sendMsg.IsUnknown() {
		var sm ActionSendMessageModel
		diags.Append(sendMsg.As(ctx, &sm, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return diags
		}
		op.OperationType = 0

		mediaTypeID := sm.MediaTypeID.ValueString()
		if mediaTypeID == "" {
			mediaTypeID = "0"
		}
		op.OpMessage = &client.ActionOpMessage{
			UseDefault:  boolToInt64(sm.UseDefaultMessage.ValueBool()),
			Subject:     sm.Subject.ValueString(),
			Message:     sm.Message.ValueString(),
			MediaTypeID: mediaTypeID,
		}

		op.OpMessageGrp = []client.ActionOpRecipientGroup{}
		if !sm.UserGroupIDs.IsNull() && !sm.UserGroupIDs.IsUnknown() {
			var grpIDs []string
			diags.Append(sm.UserGroupIDs.ElementsAs(ctx, &grpIDs, false)...)
			if !diags.HasError() {
				op.OpMessageGrp = make([]client.ActionOpRecipientGroup, len(grpIDs))
				for i, id := range grpIDs {
					op.OpMessageGrp[i] = client.ActionOpRecipientGroup{UserGroupID: id}
				}
			}
		}

		op.OpMessageUsr = []client.ActionOpRecipientUser{}
		if !sm.UserIDs.IsNull() && !sm.UserIDs.IsUnknown() {
			var userIDs []string
			diags.Append(sm.UserIDs.ElementsAs(ctx, &userIDs, false)...)
			if !diags.HasError() {
				op.OpMessageUsr = make([]client.ActionOpRecipientUser, len(userIDs))
				for i, id := range userIDs {
					op.OpMessageUsr[i] = client.ActionOpRecipientUser{UserID: id}
				}
			}
		}
		return diags
	}

	if !remoteCmd.IsNull() && !remoteCmd.IsUnknown() {
		var rc ActionRemoteCommandModel
		diags.Append(remoteCmd.As(ctx, &rc, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return diags
		}
		op.OperationType = 1
		op.OpCommand = &client.ActionOpCommand{}
		diags.Append(modelToClientRemoteCommandTargets(ctx, rc, op)...)
	}
	return diags
}

func modelToClientRemoteCommand(ctx context.Context, rc ActionRemoteCommandModel, op *client.ActionOpCommand) diag.Diagnostics {
	var diags diag.Diagnostics

	if !rc.CustomScript.IsNull() && !rc.CustomScript.IsUnknown() {
		var cs ActionCustomScriptModel
		diags.Append(rc.CustomScript.As(ctx, &cs, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			op.Type = 0
			op.Command = cs.Command.ValueString()
			op.ExecuteOn = actionExecuteOnMap[cs.ExecuteOn.ValueString()]
		}
	} else if !rc.IPMI.IsNull() && !rc.IPMI.IsUnknown() {
		var ipmi ActionIPMIModel
		diags.Append(rc.IPMI.As(ctx, &ipmi, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			op.Type = 1
			op.Command = ipmi.Command.ValueString()
		}
	} else if !rc.SSH.IsNull() && !rc.SSH.IsUnknown() {
		var ssh ActionSSHModel
		diags.Append(rc.SSH.As(ctx, &ssh, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			op.Type = 2
			op.Command = ssh.Command.ValueString()
			op.AuthType = actionSSHAuthTypeMap[ssh.AuthType.ValueString()]
			op.Username = ssh.Username.ValueString()
			op.Password = ssh.Password.ValueString()
			op.PublicKey = ssh.PublicKey.ValueString()
			op.PrivateKey = ssh.PrivateKey.ValueString()
			op.Port = ssh.Port.ValueString()
		}
	} else if !rc.Telnet.IsNull() && !rc.Telnet.IsUnknown() {
		var tel ActionTelnetModel
		diags.Append(rc.Telnet.As(ctx, &tel, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			op.Type = 3
			op.Command = tel.Command.ValueString()
			op.Username = tel.Username.ValueString()
			op.Password = tel.Password.ValueString()
			op.Port = tel.Port.ValueString()
		}
	} else if !rc.GlobalScript.IsNull() && !rc.GlobalScript.IsUnknown() {
		var gs ActionGlobalScriptModel
		diags.Append(rc.GlobalScript.As(ctx, &gs, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			op.Type = 4
			op.ScriptID = gs.ScriptID.ValueString()
		}
	}

	return diags
}

// modelToClientRemoteCommandTargets fills op.OpCommand (type and params) and
// op.OpCommandHst / op.OpCommandGrp (host/group targets) from ActionRemoteCommandModel.
func modelToClientRemoteCommandTargets(ctx context.Context, rc ActionRemoteCommandModel, parentOp *client.ActionOperation) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(modelToClientRemoteCommand(ctx, rc, parentOp.OpCommand)...)

	parentOp.OpCommandHst = []client.ActionOpCommandHost{}
	if rc.CurrentHost.ValueBool() {
		parentOp.OpCommandHst = append(parentOp.OpCommandHst, client.ActionOpCommandHost{HostID: "0"})
	}
	if !rc.HostIDs.IsNull() && !rc.HostIDs.IsUnknown() {
		var hids []string
		diags.Append(rc.HostIDs.ElementsAs(ctx, &hids, false)...)
		if !diags.HasError() {
			for _, id := range hids {
				parentOp.OpCommandHst = append(parentOp.OpCommandHst, client.ActionOpCommandHost{HostID: id})
			}
		}
	}

	parentOp.OpCommandGrp = []client.ActionOpCommandGroup{}
	if !rc.HostGroupIDs.IsNull() && !rc.HostGroupIDs.IsUnknown() {
		var gids []string
		diags.Append(rc.HostGroupIDs.ElementsAs(ctx, &gids, false)...)
		if !diags.HasError() {
			for _, id := range gids {
				parentOp.OpCommandGrp = append(parentOp.OpCommandGrp, client.ActionOpCommandGroup{GroupID: id})
			}
		}
	}

	return diags
}

func clientActionToModel(ctx context.Context, a client.Action, data *ActionTriggerResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Name = types.StringValue(a.Name)
	data.EscalationPeriod = types.StringValue(a.EscPeriod)
	data.PauseSuppressed = types.BoolValue(a.PauseSuppressed == 1)
	data.NotifyIfCanceled = types.BoolValue(a.NotifyIfCanceled == 1)
	status, ok := actionStatusReverseMap[a.Status]
	if !ok {
		diags.AddError("Unknown action status", fmt.Sprintf("Unrecognized status %d from API.", a.Status))
		return diags
	}
	data.Status = types.StringValue(status)

	// filter
	filterObj, d := clientFilterToModel(ctx, a.Filter)
	diags.Append(d...)
	if !d.HasError() {
		data.Filter = filterObj
	}

	// operations
	if !data.Operations.IsNull() || len(a.Operations) > 0 {
		opVals := make([]attr.Value, len(a.Operations))
		for i, op := range a.Operations {
			opObj, d := clientOpToModel(ctx, op)
			diags.Append(d...)
			opVals[i] = opObj
		}
		opList, d := types.ListValue(types.ObjectType{AttrTypes: actionOperationAttrTypes}, opVals)
		diags.Append(d...)
		if !d.HasError() {
			data.Operations = opList
		}
	}

	// recovery_operations
	if !data.RecoveryOperations.IsNull() || len(a.RecoveryOperations) > 0 {
		recVals := make([]attr.Value, len(a.RecoveryOperations))
		for i, op := range a.RecoveryOperations {
			recObj, d := clientRecoveryOpToModel(ctx, op)
			diags.Append(d...)
			recVals[i] = recObj
		}
		recList, d := types.ListValue(types.ObjectType{AttrTypes: actionRecoveryOpAttrTypes}, recVals)
		diags.Append(d...)
		if !d.HasError() {
			data.RecoveryOperations = recList
		}
	}

	// update_operations
	if !data.UpdateOperations.IsNull() || len(a.UpdateOperations) > 0 {
		updVals := make([]attr.Value, len(a.UpdateOperations))
		for i, op := range a.UpdateOperations {
			updObj, d := clientRecoveryOpToModel(ctx, op)
			diags.Append(d...)
			updVals[i] = updObj
		}
		updList, d := types.ListValue(types.ObjectType{AttrTypes: actionRecoveryOpAttrTypes}, updVals)
		diags.Append(d...)
		if !d.HasError() {
			data.UpdateOperations = updList
		}
	}

	return diags
}

func clientFilterToModel(_ context.Context, f client.ActionFilter) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	evalType, ok := actionEvalTypeReverseMap[f.EvalType]
	if !ok {
		diags.AddError("Unknown evaltype", fmt.Sprintf("Unrecognized evaltype %d from API.", f.EvalType))
		return types.ObjectNull(actionFilterAttrTypes), diags
	}

	customExpr := f.EvalType == 3
	condVals := make([]attr.Value, len(f.Conditions))
	for i, c := range f.Conditions {
		ct, ok := actionConditionTypeReverseMap[c.ConditionType]
		if !ok {
			diags.AddError("Unknown condition type", fmt.Sprintf("Unrecognized conditiontype %d from API.", c.ConditionType))
			return types.ObjectNull(actionFilterAttrTypes), diags
		}
		op, ok := actionOperatorReverseMap[c.Operator]
		if !ok {
			diags.AddError("Unknown operator", fmt.Sprintf("Unrecognized operator %d from API.", c.Operator))
			return types.ObjectNull(actionFilterAttrTypes), diags
		}
		label := ""
		if customExpr {
			label = c.FormulaID
		}
		condObj, d := types.ObjectValue(actionConditionAttrTypes, map[string]attr.Value{
			"condition_type": types.StringValue(ct),
			"operator":       types.StringValue(op),
			"value":          types.StringValue(c.Value),
			"value2":         types.StringValue(c.Value2),
			"label":          types.StringValue(label),
		})
		diags.Append(d...)
		condVals[i] = condObj
	}

	condList, d := types.ListValue(types.ObjectType{AttrTypes: actionConditionAttrTypes}, condVals)
	diags.Append(d...)

	formula := ""
	if customExpr {
		formula = f.Formula
	}
	filterObj, d := types.ObjectValue(actionFilterAttrTypes, map[string]attr.Value{
		"evaluation_type": types.StringValue(evalType),
		"formula":         types.StringValue(formula),
		"condition":       condList,
	})
	diags.Append(d...)
	return filterObj, diags
}

func clientOpToModel(ctx context.Context, op client.ActionOperation) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	sendMsg := types.ObjectNull(actionSendMessageAttrTypes)
	remoteCmd := types.ObjectNull(actionRemoteCommandAttrTypes)

	switch op.OperationType {
	case 0:
		sm, d := clientSendMessageToModel(ctx, op)
		diags.Append(d...)
		sendMsg = sm
	case 1:
		rc, d := clientRemoteCommandToModel(ctx, op)
		diags.Append(d...)
		remoteCmd = rc
	}

	obj, d := types.ObjectValue(actionOperationAttrTypes, map[string]attr.Value{
		"escalation_period":    types.StringValue(op.EscPeriod),
		"escalation_step_from": types.Int64Value(op.EscStepFrom),
		"escalation_step_to":   types.Int64Value(op.EscStepTo),
		"send_message":         sendMsg,
		"remote_command":       remoteCmd,
	})
	diags.Append(d...)
	return obj, diags
}

func clientRecoveryOpToModel(ctx context.Context, op client.ActionOperation) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	notifyAll := false
	sendMsg := types.ObjectNull(actionSendMessageAttrTypes)
	remoteCmd := types.ObjectNull(actionRemoteCommandAttrTypes)

	switch op.OperationType {
	case 0:
		sm, d := clientSendMessageToModel(ctx, op)
		diags.Append(d...)
		sendMsg = sm
	case 1:
		rc, d := clientRemoteCommandToModel(ctx, op)
		diags.Append(d...)
		remoteCmd = rc
	case 11:
		notifyAll = true
	}

	obj, d := types.ObjectValue(actionRecoveryOpAttrTypes, map[string]attr.Value{
		"notify_all_involved": types.BoolValue(notifyAll),
		"send_message":        sendMsg,
		"remote_command":      remoteCmd,
	})
	diags.Append(d...)
	return obj, diags
}

func clientSendMessageToModel(_ context.Context, op client.ActionOperation) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if op.OpMessage == nil {
		return types.ObjectNull(actionSendMessageAttrTypes), diags
	}

	grpVals := make([]attr.Value, len(op.OpMessageGrp))
	for i, g := range op.OpMessageGrp {
		grpVals[i] = types.StringValue(g.UserGroupID)
	}
	grpSet, d := types.SetValue(types.StringType, grpVals)
	diags.Append(d...)

	usrVals := make([]attr.Value, len(op.OpMessageUsr))
	for i, u := range op.OpMessageUsr {
		usrVals[i] = types.StringValue(u.UserID)
	}
	usrSet, d := types.SetValue(types.StringType, usrVals)
	diags.Append(d...)

	obj, d := types.ObjectValue(actionSendMessageAttrTypes, map[string]attr.Value{
		"use_default_message": types.BoolValue(op.OpMessage.UseDefault == 1),
		"subject":             types.StringValue(op.OpMessage.Subject),
		"message":             types.StringValue(op.OpMessage.Message),
		"media_type_id":       types.StringValue(op.OpMessage.MediaTypeID),
		"user_group_ids":      grpSet,
		"user_ids":            usrSet,
	})
	diags.Append(d...)
	return obj, diags
}

func clientRemoteCommandToModel(_ context.Context, op client.ActionOperation) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if op.OpCommand == nil {
		return types.ObjectNull(actionRemoteCommandAttrTypes), diags
	}

	customScript := types.ObjectNull(actionCustomScriptAttrTypes)
	ipmi := types.ObjectNull(actionIPMIAttrTypes)
	ssh := types.ObjectNull(actionSSHAttrTypes)
	telnet := types.ObjectNull(actionTelnetAttrTypes)
	globalScript := types.ObjectNull(actionGlobalScriptAttrTypes)

	switch op.OpCommand.Type {
	case 0:
		eo, ok := actionExecuteOnReverseMap[op.OpCommand.ExecuteOn]
		if !ok {
			diags.AddError("Unknown execute_on", fmt.Sprintf("Unrecognized execute_on %d from API.", op.OpCommand.ExecuteOn))
		} else {
			obj, d := types.ObjectValue(actionCustomScriptAttrTypes, map[string]attr.Value{
				"command":    types.StringValue(op.OpCommand.Command),
				"execute_on": types.StringValue(eo),
			})
			diags.Append(d...)
			customScript = obj
		}
	case 1:
		obj, d := types.ObjectValue(actionIPMIAttrTypes, map[string]attr.Value{
			"command": types.StringValue(op.OpCommand.Command),
		})
		diags.Append(d...)
		ipmi = obj
	case 2:
		at, ok := actionSSHAuthTypeReverseMap[op.OpCommand.AuthType]
		if !ok {
			diags.AddError("Unknown authtype", fmt.Sprintf("Unrecognized authtype %d from API.", op.OpCommand.AuthType))
		} else {
			obj, d := types.ObjectValue(actionSSHAttrTypes, map[string]attr.Value{
				"command":     types.StringValue(op.OpCommand.Command),
				"authtype":    types.StringValue(at),
				"username":    types.StringValue(op.OpCommand.Username),
				"password":    types.StringValue(op.OpCommand.Password),
				"public_key":  types.StringValue(op.OpCommand.PublicKey),
				"private_key": types.StringValue(op.OpCommand.PrivateKey),
				"port":        types.StringValue(op.OpCommand.Port),
			})
			diags.Append(d...)
			ssh = obj
		}
	case 3:
		obj, d := types.ObjectValue(actionTelnetAttrTypes, map[string]attr.Value{
			"command":  types.StringValue(op.OpCommand.Command),
			"username": types.StringValue(op.OpCommand.Username),
			"password": types.StringValue(op.OpCommand.Password),
			"port":     types.StringValue(op.OpCommand.Port),
		})
		diags.Append(d...)
		telnet = obj
	case 4:
		obj, d := types.ObjectValue(actionGlobalScriptAttrTypes, map[string]attr.Value{
			"script_id": types.StringValue(op.OpCommand.ScriptID),
		})
		diags.Append(d...)
		globalScript = obj
	}

	// targets
	currentHost := false
	hstIDs := []attr.Value{}
	grpIDs := []attr.Value{}
	for _, h := range op.OpCommandHst {
		if h.HostID == "0" {
			currentHost = true
		} else {
			hstIDs = append(hstIDs, types.StringValue(h.HostID))
		}
	}
	for _, g := range op.OpCommandGrp {
		grpIDs = append(grpIDs, types.StringValue(g.GroupID))
	}
	hstSet, d := types.SetValue(types.StringType, hstIDs)
	diags.Append(d...)
	grpSet, d := types.SetValue(types.StringType, grpIDs)
	diags.Append(d...)

	obj, d := types.ObjectValue(actionRemoteCommandAttrTypes, map[string]attr.Value{
		"current_host":   types.BoolValue(currentHost),
		"host_ids":       hstSet,
		"host_group_ids": grpSet,
		"custom_script":  customScript,
		"ipmi":           ipmi,
		"ssh":            ssh,
		"telnet":         telnet,
		"global_script":  globalScript,
	})
	diags.Append(d...)
	return obj, diags
}

// boolToInt64 converts a bool to the 0/1 integer representation used by Zabbix.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
