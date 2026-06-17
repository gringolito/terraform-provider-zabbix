package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/clienttest"
	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// ---- Acceptance tests ----

func TestAccActionTriggerDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-at-ds"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionTriggerDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("enabled"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("escalation_period"),
						knownvalue.StringExact("1h"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("pause_suppressed"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("notify_if_canceled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("filter").AtMapKey("evaluation_type"),
						knownvalue.StringExact("and_or"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("operations"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("recovery_operations"),
						knownvalue.ListSizeExact(0),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("update_operations"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}

func TestAccActionTriggerDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-at-ds-name"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionTriggerDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("enabled"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_action_trigger.test",
						tfjsonpath.New("escalation_period"),
						knownvalue.StringExact("1h"),
					),
				},
			},
		},
	})
}

// ---- Unit tests ----

func TestActionTriggerDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeActionTriggerDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildActionTriggerDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when no lookup key is set")
	}
}

func TestActionTriggerDataSource_NotFoundError(t *testing.T) {
	fake := &clienttest.TestClient{Response: []map[string]any{}}

	ds := newFakeActionTriggerDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildActionTriggerDataSourceConfig(t, "999", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for not-found")
	}
}

func TestActionTriggerDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			actionRPCResponse("1", "My Action"),
			actionRPCResponse("2", "My Action"),
		},
	}

	ds := newFakeActionTriggerDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildActionTriggerDataSourceConfig(t, "", "My Action")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple trigger actions found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple trigger actions found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestActionTriggerDataSource_SuccessfulRead(t *testing.T) {
	ctx := context.Background()
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			actionRPCResponseWithConditionAndOp("42", "Notify on Problem"),
		},
	}

	ds := newFakeActionTriggerDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildActionTriggerDataSourceConfig(t, "42", "")}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewActionTriggerDataSource().Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics)
	}

	var got provider.ActionTriggerDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state.Get: %s", diags)
	}

	if got.ID.ValueString() != "42" {
		t.Errorf("id: got %q, want %q", got.ID.ValueString(), "42")
	}
	if got.Name.ValueString() != "Notify on Problem" {
		t.Errorf("name: got %q, want %q", got.Name.ValueString(), "Notify on Problem")
	}
	if got.Status.ValueString() != "enabled" {
		t.Errorf("status: got %q, want %q", got.Status.ValueString(), "enabled")
	}
	if got.EscalationPeriod.ValueString() != "1h" {
		t.Errorf("escalation_period: got %q, want %q", got.EscalationPeriod.ValueString(), "1h")
	}
	if !got.PauseSuppressed.ValueBool() {
		t.Error("pause_suppressed: expected true")
	}
	if !got.NotifyIfCanceled.ValueBool() {
		t.Error("notify_if_canceled: expected true")
	}
	if got.Filter.IsNull() {
		t.Error("filter: expected non-null")
	}
	if got.Operations.IsNull() || got.Operations.IsUnknown() {
		t.Fatal("operations: expected non-null list")
	}
	if len(got.Operations.Elements()) != 1 {
		t.Errorf("operations: got %d elements, want 1", len(got.Operations.Elements()))
	}
	if len(got.RecoveryOperations.Elements()) != 0 {
		t.Errorf("recovery_operations: got %d elements, want 0", len(got.RecoveryOperations.Elements()))
	}
	if len(got.UpdateOperations.Elements()) != 0 {
		t.Errorf("update_operations: got %d elements, want 0", len(got.UpdateOperations.Elements()))
	}
}

// ---- helpers ----

func actionRPCResponse(id, name string) map[string]any {
	return map[string]any{
		"actionid":           id,
		"name":               name,
		"eventsource":        "0",
		"status":             "0",
		"esc_period":         "1h",
		"pause_suppressed":   "1",
		"notify_if_canceled": "1",
		"def_shortdata":      "",
		"def_longdata":       "",
		"filter": map[string]any{
			"evaltype":   "0",
			"formula":    "",
			"conditions": []any{},
		},
		"operations":          []any{},
		"recovery_operations": []any{},
		"update_operations":   []any{},
	}
}

func actionRPCResponseWithConditionAndOp(id, name string) map[string]any {
	return map[string]any{
		"actionid":           id,
		"name":               name,
		"eventsource":        "0",
		"status":             "0",
		"esc_period":         "1h",
		"pause_suppressed":   "1",
		"notify_if_canceled": "1",
		"def_shortdata":      "",
		"def_longdata":       "",
		"filter": map[string]any{
			"evaltype": "0",
			"formula":  "",
			"conditions": []any{
				map[string]any{
					"conditiontype": "4",
					"operator":      "5",
					"value":         "3",
					"value2":        "",
					"formulaid":     "A",
				},
			},
		},
		"operations": []any{
			map[string]any{
				"operationtype": "0",
				"esc_period":    "0",
				"esc_step_from": "1",
				"esc_step_to":   "1",
				"opmessage": map[string]any{
					"default_msg": "1",
					"subject":     "",
					"message":     "",
					"mediatypeid": "0",
				},
				"opmessage_grp": []any{
					map[string]any{"usrgrpid": "7"},
				},
				"opmessage_usr": []any{},
				"opcommand_hst": []any{},
				"opcommand_grp": []any{},
			},
		},
		"recovery_operations": []any{},
		"update_operations":   []any{},
	}
}

func newFakeActionTriggerDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewActionTriggerDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildActionTriggerDataSourceConfig(t *testing.T, id, name string) tfsdk.Config {
	t.Helper()

	null := func(ty tftypes.Type) tftypes.Value { return tftypes.NewValue(ty, nil) }
	toStr := func(s string) tftypes.Value {
		if s == "" {
			return null(tftypes.String)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	condType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"condition_type": tftypes.String,
		"operator":       tftypes.String,
		"value":          tftypes.String,
		"value2":         tftypes.String,
		"label":          tftypes.String,
	}}
	filterType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"evaluation_type": tftypes.String,
		"formula":         tftypes.String,
		"condition":       tftypes.List{ElementType: condType},
	}}

	sendMsgType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"use_default_message": tftypes.Bool,
		"subject":             tftypes.String,
		"message":             tftypes.String,
		"media_type_id":       tftypes.String,
		"user_group_ids":      tftypes.Set{ElementType: tftypes.String},
		"user_ids":            tftypes.Set{ElementType: tftypes.String},
	}}
	customScriptType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"command":    tftypes.String,
		"execute_on": tftypes.String,
	}}
	ipmiType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"command": tftypes.String,
	}}
	sshType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"command":     tftypes.String,
		"authtype":    tftypes.String,
		"username":    tftypes.String,
		"password":    tftypes.String,
		"public_key":  tftypes.String,
		"private_key": tftypes.String,
		"port":        tftypes.String,
	}}
	telnetType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"command":  tftypes.String,
		"username": tftypes.String,
		"password": tftypes.String,
		"port":     tftypes.String,
	}}
	globalScriptType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"script_id": tftypes.String,
	}}
	remoteCmdType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"current_host":   tftypes.Bool,
		"host_ids":       tftypes.Set{ElementType: tftypes.String},
		"host_group_ids": tftypes.Set{ElementType: tftypes.String},
		"custom_script":  customScriptType,
		"ipmi":           ipmiType,
		"ssh":            sshType,
		"telnet":         telnetType,
		"global_script":  globalScriptType,
	}}
	opType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"escalation_period":    tftypes.String,
		"escalation_step_from": tftypes.Number,
		"escalation_step_to":   tftypes.Number,
		"send_message":         sendMsgType,
		"remote_command":       remoteCmdType,
	}}
	recoveryOpType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"notify_all_involved": tftypes.Bool,
		"send_message":        sendMsgType,
		"remote_command":      remoteCmdType,
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewActionTriggerDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":                  tftypes.String,
				"name":                tftypes.String,
				"status":              tftypes.String,
				"escalation_period":   tftypes.String,
				"pause_suppressed":    tftypes.Bool,
				"notify_if_canceled":  tftypes.Bool,
				"filter":              filterType,
				"operations":          tftypes.List{ElementType: opType},
				"recovery_operations": tftypes.List{ElementType: recoveryOpType},
				"update_operations":   tftypes.List{ElementType: recoveryOpType},
			},
		}, map[string]tftypes.Value{
			"id":                  toStr(id),
			"name":                toStr(name),
			"status":              null(tftypes.String),
			"escalation_period":   null(tftypes.String),
			"pause_suppressed":    null(tftypes.Bool),
			"notify_if_canceled":  null(tftypes.Bool),
			"filter":              null(filterType),
			"operations":          null(tftypes.List{ElementType: opType}),
			"recovery_operations": null(tftypes.List{ElementType: recoveryOpType}),
			"update_operations":   null(tftypes.List{ElementType: recoveryOpType}),
		}),
		Schema: schemaResp.Schema,
	}
}

// ---- config helpers ----

func testAccActionTriggerDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return testAccActionTriggerBase(cfg) + fmt.Sprintf(`
resource "zabbix_action_trigger" "seed" {
  name              = %[1]q
  escalation_period = "1h"

  filter {
    evaluation_type = "and_or"
    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
    }
  }

  operations {
    escalation_step_from = 1
    escalation_step_to   = 1
    escalation_period    = "0"

    send_message {
      use_default_message = true
      user_group_ids      = [data.zabbix_user_group.admins.id]
    }
  }
}

data "zabbix_action_trigger" "test" {
  id = zabbix_action_trigger.seed.id
}
`, name)
}

func testAccActionTriggerDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return testAccActionTriggerBase(cfg) + fmt.Sprintf(`
resource "zabbix_action_trigger" "seed" {
  name              = %[1]q
  escalation_period = "1h"

  filter {
    evaluation_type = "and_or"
    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
    }
  }

  operations {
    escalation_step_from = 1
    escalation_step_to   = 1
    escalation_period    = "0"

    send_message {
      use_default_message = true
      user_group_ids      = [data.zabbix_user_group.admins.id]
    }
  }
}

data "zabbix_action_trigger" "test" {
  depends_on = [zabbix_action_trigger.seed]
  name       = %[1]q
}
`, name)
}
