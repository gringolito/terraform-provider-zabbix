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

	schemaResp := &datasource.SchemaResponse{}
	provider.NewActionTriggerDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":     tftypes.String,
				"name":   tftypes.String,
				"status": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"id":     toStr(id),
			"name":   toStr(name),
			"status": null(tftypes.String),
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
