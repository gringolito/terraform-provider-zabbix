package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// ---- Acceptance tests ----

func TestAccActionTriggerResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-at"
	updated := cfg.NamePrefix + "-at-upd"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccActionTriggerResourceConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("enabled"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("escalation_period"),
						knownvalue.StringExact("1h"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("pause_suppressed"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("notify_if_canceled"),
						knownvalue.Bool(true),
					),
				},
			},
			// Update name and disable
			{
				Config: testAccActionTriggerResourceConfigDisabled(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updated),
					),
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("disabled"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccActionTriggerResource_WithRecoveryOperation(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-at-rec"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionTriggerResourceWithRecoveryConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccActionTriggerResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-at-imp"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionTriggerResourceConfig(cfg, name),
			},
			{
				ResourceName:      "zabbix_action_trigger.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccActionTriggerResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-at-drift"

	var capturedID string

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccActionTriggerResourceConfig(cfg, name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_action_trigger.test"]
						if rs == nil {
							return fmt.Errorf("resource not found in state")
						}
						capturedID = rs.Primary.ID
						return nil
					},
				),
			},
			// Delete out-of-band, expect recreate
			{
				PreConfig: func() {
					c, err := client.New(context.Background(), cfg.URL, cfg.Token)
					if err != nil {
						t.Fatalf("drift PreConfig: client.New: %v", err)
					}
					if err := client.ActionDelete(context.Background(), c, capturedID); err != nil {
						t.Fatalf("drift PreConfig: ActionDelete: %v", err)
					}
				},
				Config: testAccActionTriggerResourceConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_action_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

// ---- Unit tests (ConfigValidators) ----

func TestActionTriggerResource_CustomExpressionRequiresFormula(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionTriggerCustomExprNoFormulaConfig(cfg),
				ExpectError: regexp.MustCompile(`formula`),
			},
		},
	})
}

func TestActionTriggerResource_CustomExpressionForbidsFormulaWhenNotCustom(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionTriggerNonCustomExprWithFormulaConfig(cfg),
				ExpectError: regexp.MustCompile(`formula`),
			},
		},
	})
}

func TestActionTriggerResource_CustomExpressionConditionRequiresLabel(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionTriggerCustomExprConditionNoLabelConfig(cfg),
				ExpectError: regexp.MustCompile(`label`),
			},
		},
	})
}

func TestActionTriggerResource_NonCustomExpressionConditionForbidsLabel(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionTriggerNonCustomConditionWithLabelConfig(cfg),
				ExpectError: regexp.MustCompile(`label`),
			},
		},
	})
}

func TestActionTriggerResource_SendMessageUseDefaultForbidsSubjectMessage(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionTriggerSendMessageUseDefaultWithSubjectConfig(cfg),
				ExpectError: regexp.MustCompile(`subject`),
			},
		},
	})
}

func TestActionTriggerResource_SendMessageNotDefaultRequiresSubjectMessage(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccActionTriggerSendMessageNotDefaultNoSubjectConfig(cfg),
				ExpectError: regexp.MustCompile(`subject`),
			},
		},
	})
}

// ---- config helpers ----

func testAccActionTriggerBase(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_user_group" "admins" {
  name = "Zabbix administrators"
}
`, cfg.URL, cfg.Token)
}

func testAccActionTriggerResourceConfig(cfg *testhelper.Config, name string) string {
	return testAccActionTriggerBase(cfg) + fmt.Sprintf(`
resource "zabbix_action_trigger" "test" {
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
`, name)
}

func testAccActionTriggerResourceConfigDisabled(cfg *testhelper.Config, name string) string {
	return testAccActionTriggerBase(cfg) + fmt.Sprintf(`
resource "zabbix_action_trigger" "test" {
  name              = %[1]q
  status            = "disabled"
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
`, name)
}

func testAccActionTriggerResourceWithRecoveryConfig(cfg *testhelper.Config, name string) string {
	return testAccActionTriggerBase(cfg) + fmt.Sprintf(`
resource "zabbix_action_trigger" "test" {
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

  recovery_operations {
    notify_all_involved = true
  }
}
`, name)
}

// ---- unit test config helpers ----

func testAccActionTriggerCustomExprNoFormulaConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_action_trigger" "test" {
  name              = "test"
  escalation_period = "1h"

  filter {
    evaluation_type = "custom_expression"
    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
      label          = "A"
    }
  }
}
`, cfg.URL, cfg.Token)
}

func testAccActionTriggerNonCustomExprWithFormulaConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_action_trigger" "test" {
  name              = "test"
  escalation_period = "1h"

  filter {
    evaluation_type = "and_or"
    formula         = "{A}"
    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
    }
  }
}
`, cfg.URL, cfg.Token)
}

func testAccActionTriggerCustomExprConditionNoLabelConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_action_trigger" "test" {
  name              = "test"
  escalation_period = "1h"

  filter {
    evaluation_type = "custom_expression"
    formula         = "{A}"
    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
    }
  }
}
`, cfg.URL, cfg.Token)
}

func testAccActionTriggerNonCustomConditionWithLabelConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_action_trigger" "test" {
  name              = "test"
  escalation_period = "1h"

  filter {
    evaluation_type = "and_or"
    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
      label          = "A"
    }
  }
}
`, cfg.URL, cfg.Token)
}

func testAccActionTriggerSendMessageUseDefaultWithSubjectConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_action_trigger" "test" {
  name              = "test"
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

    send_message {
      use_default_message = true
      subject             = "Problem: {EVENT.NAME}"
    }
  }
}
`, cfg.URL, cfg.Token)
}

func testAccActionTriggerSendMessageNotDefaultNoSubjectConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_action_trigger" "test" {
  name              = "test"
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

    send_message {
      use_default_message = false
      message             = "something"
    }
  }
}
`, cfg.URL, cfg.Token)
}
