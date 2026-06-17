package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ActionTriggerDataSource{}
var _ datasource.DataSourceWithConfigure = &ActionTriggerDataSource{}

func NewActionTriggerDataSource() datasource.DataSource {
	return &ActionTriggerDataSource{}
}

type ActionTriggerDataSource struct {
	client client.Client
}

type ActionTriggerDataSourceModel struct {
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

func (d *ActionTriggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_trigger"
}

func (d *ActionTriggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	conditionAttrs := map[string]schema.Attribute{
		"condition_type": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Type of condition.",
		},
		"operator": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Comparison operator.",
		},
		"value": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Value to compare against.",
		},
		"value2": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Secondary value (used by some condition types).",
		},
		"label": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Condition label used in custom expression formulas (e.g. `A`, `B`).",
		},
	}

	sendMessageAttrs := map[string]schema.Attribute{
		"use_default_message": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the media type's default subject/message are used.",
		},
		"subject": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Custom message subject.",
		},
		"message": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Custom message body.",
		},
		"media_type_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of the media type to use.",
		},
		"user_group_ids": schema.SetAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Set of user group IDs to send the message to.",
		},
		"user_ids": schema.SetAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Set of user IDs to send the message to.",
		},
	}

	remoteCommandAttrs := map[string]schema.Attribute{
		"current_host": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Run the command on the host that triggered the event.",
		},
		"host_ids": schema.SetAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Set of host IDs to run the command on.",
		},
		"host_group_ids": schema.SetAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Set of host group IDs to run the command on.",
		},
		"custom_script": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Execute a custom script.",
			Attributes: map[string]schema.Attribute{
				"command": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Script body to execute.",
				},
				"execute_on": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Where to execute the script. One of: `agent`, `server_or_proxy`, `server`.",
				},
			},
		},
		"ipmi": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Execute an IPMI command.",
			Attributes: map[string]schema.Attribute{
				"command": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "IPMI command to execute.",
				},
			},
		},
		"ssh": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Execute a command over SSH.",
			Attributes: map[string]schema.Attribute{
				"command": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Command to execute over SSH.",
				},
				"authtype": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Authentication type. One of: `password`, `public_key`.",
				},
				"username": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "SSH username.",
				},
				"password": schema.StringAttribute{
					Computed:            true,
					Sensitive:           true,
					MarkdownDescription: "SSH password (used when `authtype = \"password\"`).",
				},
				"public_key": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Public key for SSH authentication (used when `authtype = \"public_key\"`).",
				},
				"private_key": schema.StringAttribute{
					Computed:            true,
					Sensitive:           true,
					MarkdownDescription: "Private key for SSH authentication (used when `authtype = \"public_key\"`).",
				},
				"port": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "SSH port.",
				},
			},
		},
		"telnet": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Execute a command over Telnet.",
			Attributes: map[string]schema.Attribute{
				"command": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Command to execute over Telnet.",
				},
				"username": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Telnet username.",
				},
				"password": schema.StringAttribute{
					Computed:            true,
					Sensitive:           true,
					MarkdownDescription: "Telnet password.",
				},
				"port": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "Telnet port.",
				},
			},
		},
		"global_script": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Run a Zabbix global script.",
			Attributes: map[string]schema.Attribute{
				"script_id": schema.StringAttribute{
					Computed:            true,
					MarkdownDescription: "ID of the Zabbix global script to run.",
				},
			},
		},
	}

	operationAttrs := map[string]schema.Attribute{
		"escalation_period": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Escalation period override for this operation step.",
		},
		"escalation_step_from": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "First escalation step this operation applies to.",
		},
		"escalation_step_to": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Last escalation step this operation applies to (0 = last step).",
		},
		"send_message": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Send a notification message.",
			Attributes:          sendMessageAttrs,
		},
		"remote_command": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Execute a remote command.",
			Attributes:          remoteCommandAttrs,
		},
	}

	recoveryOpAttrs := map[string]schema.Attribute{
		"notify_all_involved": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Notify all users who received the problem notification.",
		},
		"send_message": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Send a notification message.",
			Attributes:          sendMessageAttrs,
		},
		"remote_command": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Execute a remote command.",
			Attributes:          remoteCommandAttrs,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Zabbix trigger action by `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the trigger action. Exactly one of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the trigger action. Exactly one of `id` or `name` must be set.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Status of the trigger action. One of: `enabled`, `disabled`.",
			},
			"escalation_period": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Default escalation period.",
			},
			"pause_suppressed": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether operations are paused while hosts are in a maintenance period.",
			},
			"notify_if_canceled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether recovery notifications are sent for problems canceled during maintenance.",
			},
			"filter": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Conditions that must be matched for the action to fire.",
				Attributes: map[string]schema.Attribute{
					"evaluation_type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "How conditions are evaluated. One of: `and_or`, `and`, `or`, `custom_expression`.",
					},
					"formula": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Custom expression joining condition labels.",
					},
					"condition": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Filter conditions.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: conditionAttrs,
						},
					},
				},
			},
			"operations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Problem escalation operations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: operationAttrs,
				},
			},
			"recovery_operations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Recovery operations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: recoveryOpAttrs,
				},
			},
			"update_operations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Update/acknowledge operations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: recoveryOpAttrs,
				},
			},
		},
	}
}

func (d *ActionTriggerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ActionTriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActionTriggerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != ""
	nameSet := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""

	if !idSet && !nameSet {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Exactly one of \"id\" or \"name\" must be set.",
		)
		return
	}

	var actions []client.Action

	if idSet {
		a, err := client.ActionGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger action", err.Error())
			return
		}
		if a == nil {
			resp.Diagnostics.AddError(
				"Trigger action not found",
				fmt.Sprintf("No trigger action found with id %q.", data.ID.ValueString()),
			)
			return
		}
		actions = []client.Action{*a}
	} else {
		var err error
		actions, err = client.ActionGetByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger action", err.Error())
			return
		}
		if len(actions) == 0 {
			resp.Diagnostics.AddError(
				"Trigger action not found",
				fmt.Sprintf("No trigger action found with name %q.", data.Name.ValueString()),
			)
			return
		}
		if len(actions) > 1 {
			resp.Diagnostics.AddError(
				"Multiple trigger actions found",
				fmt.Sprintf("Found %d trigger actions with name %q. Use \"id\" to disambiguate.", len(actions), data.Name.ValueString()),
			)
			return
		}
	}

	a := actions[0]
	data.ID = types.StringValue(a.ActionID)
	data.Name = types.StringValue(a.Name)
	data.EscalationPeriod = types.StringValue(a.EscPeriod)
	data.PauseSuppressed = types.BoolValue(a.PauseSuppressed == 1)
	data.NotifyIfCanceled = types.BoolValue(a.NotifyIfCanceled == 1)

	status, ok := actionStatusReverseMap[a.Status]
	if !ok {
		resp.Diagnostics.AddError("Unknown action status", fmt.Sprintf("Unrecognized status %d from API.", a.Status))
		return
	}
	data.Status = types.StringValue(status)

	filterObj, diags := clientFilterToModel(ctx, a.Filter)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Filter = filterObj

	opVals := make([]attr.Value, len(a.Operations))
	for i, op := range a.Operations {
		opObj, diags := clientOpToModel(ctx, op)
		resp.Diagnostics.Append(diags...)
		opVals[i] = opObj
	}
	opList, diags := types.ListValue(types.ObjectType{AttrTypes: actionOperationAttrTypes}, opVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Operations = opList

	recVals := make([]attr.Value, len(a.RecoveryOperations))
	for i, op := range a.RecoveryOperations {
		recObj, diags := clientRecoveryOpToModel(ctx, op)
		resp.Diagnostics.Append(diags...)
		recVals[i] = recObj
	}
	recList, diags := types.ListValue(types.ObjectType{AttrTypes: actionRecoveryOpAttrTypes}, recVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RecoveryOperations = recList

	updVals := make([]attr.Value, len(a.UpdateOperations))
	for i, op := range a.UpdateOperations {
		updObj, diags := clientRecoveryOpToModel(ctx, op)
		resp.Diagnostics.Append(diags...)
		updVals[i] = updObj
	}
	updList, diags := types.ListValue(types.ObjectType{AttrTypes: actionRecoveryOpAttrTypes}, updVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UpdateOperations = updList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
