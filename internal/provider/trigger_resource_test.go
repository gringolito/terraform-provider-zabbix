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

func TestAccTriggerResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
			},
			// Create
			{
				Config: testAccTriggerResourceConfig(cfg, tgName, tmplName, "High CPU", `last(/`+tmplName+`/`+itemKey+`)>90`, "warning"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("High CPU"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("priority"),
						knownvalue.StringExact("warning"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("enabled"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("recovery_mode"),
						knownvalue.StringExact("expression"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("manual_close"),
						knownvalue.Bool(false),
					),
				},
			},
			// Update description
			{
				Config: testAccTriggerResourceConfig(cfg, tgName, tmplName, "High CPU Updated", `last(/`+tmplName+`/`+itemKey+`)>95`, "average"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("High CPU Updated"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("priority"),
						knownvalue.StringExact("average"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTriggerResource_WithTags(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
			},
			{
				Config: testAccTriggerResourceWithTagsConfig(cfg, tgName, tmplName, itemKey),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("tags"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.ObjectExact(map[string]knownvalue.Check{
								"name":  knownvalue.StringExact("env"),
								"value": knownvalue.StringExact("prod"),
							}),
						}),
					),
				},
			},
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTriggerResource_RecoveryExpression(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
			},
			{
				Config: testAccTriggerResourceRecoveryExprConfig(cfg, tgName, tmplName, itemKey),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("recovery_mode"),
						knownvalue.StringExact("recovery_expression"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("recovery_expression"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTriggerResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
			},
			{
				Config: testAccTriggerResourceConfig(cfg, tgName, tmplName, "High CPU", `last(/`+tmplName+`/`+itemKey+`)>90`, "warning"),
			},
			{
				ResourceName:      "zabbix_trigger.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTriggerResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	var capturedID string

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
			},
			{
				Config: testAccTriggerResourceConfig(cfg, tgName, tmplName, "High CPU", `last(/`+tmplName+`/`+itemKey+`)>90`, "warning"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_trigger.test"]
						if rs == nil {
							return fmt.Errorf("zabbix_trigger.test not found in state")
						}
						capturedID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					c, err := client.New(context.Background(), cfg.URL, cfg.Token)
					if err != nil {
						t.Fatalf("drift PreConfig: client.New: %v", err)
					}
					if err := client.TriggerDelete(context.Background(), c, capturedID); err != nil {
						t.Fatalf("drift PreConfig: TriggerDelete: %v", err)
					}
				},
				Config: testAccTriggerResourceConfig(cfg, tgName, tmplName, "High CPU", `last(/`+tmplName+`/`+itemKey+`)>90`, "warning"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey),
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

// ---- Unit tests ----

func TestTriggerResource_RecoveryExpressionRequiredWhenModeSet(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTriggerResourceRecoveryExprMissingConfig(cfg),
				ExpectError: regexp.MustCompile(`recovery_expression`),
			},
		},
	})
}

func TestTriggerResource_RecoveryExpressionMustNotBeSetWhenModeIsExpression(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTriggerResourceRecoveryExprSetWhenModeExpressionConfig(cfg),
				ExpectError: regexp.MustCompile(`recovery_expression`),
			},
		},
	})
}

// ---- config helpers ----

func testAccTriggerImportSetup(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "test" {
  name = %[3]q

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.zabbix_template_import.test]
    }
  }
}

action "zabbix_template_import" "test" {
  config {
    source = %[4]q
    format = "xml"
  }
}
`, cfg.URL, cfg.Token, tgName, xmlExportWithItems(tmplName, tgName, itemKey))
}

func testAccTriggerResourceConfig(cfg *testhelper.Config, tgName, tmplName, description, expression, priority string) string {
	return testAccTriggerImportSetup(cfg, tgName, tmplName, "system.cpu.util") + fmt.Sprintf(`
resource "zabbix_trigger" "test" {
  depends_on  = [zabbix_template_group.test]
  description = %[1]q
  expression  = %[2]q
  priority    = %[3]q
}
`, description, expression, priority)
}

func testAccTriggerResourceWithTagsConfig(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	expr := fmt.Sprintf(`last(/%s/%s)>90`, tmplName, itemKey)
	return testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey) + fmt.Sprintf(`
resource "zabbix_trigger" "test" {
  depends_on  = [zabbix_template_group.test]
  description = "High CPU"
  expression  = %[1]q
  priority    = "warning"
  tags = [
    { name = "env", value = "prod" },
  ]
}
`, expr)
}

func testAccTriggerResourceRecoveryExprConfig(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	expr := fmt.Sprintf(`last(/%s/%s)>90`, tmplName, itemKey)
	recExpr := fmt.Sprintf(`last(/%s/%s)<70`, tmplName, itemKey)
	return testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey) + fmt.Sprintf(`
resource "zabbix_trigger" "test" {
  depends_on           = [zabbix_template_group.test]
  description          = "High CPU"
  expression           = %[1]q
  priority             = "warning"
  recovery_mode        = "recovery_expression"
  recovery_expression  = %[2]q
}
`, expr, recExpr)
}

func testAccTriggerResourceRecoveryExprMissingConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_trigger" "test" {
  description   = "Test"
  expression    = "last(/h/k)>0"
  priority      = "warning"
  recovery_mode = "recovery_expression"
}
`, cfg.URL, cfg.Token)
}

func testAccTriggerResourceRecoveryExprSetWhenModeExpressionConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_trigger" "test" {
  description         = "Test"
  expression          = "last(/h/k)>0"
  priority            = "warning"
  recovery_mode       = "expression"
  recovery_expression = "last(/h/k)<5"
}
`, cfg.URL, cfg.Token)
}

// xmlExportWithItems returns a minimal Zabbix 7.0 XML export with a template
// containing one item, suitable as a fixture for trigger acceptance tests.
// The <template_groups> section is intentionally omitted: the group is already
// managed by zabbix_template_group.test, and including it with a random UUID
// causes Zabbix to fail on a name-uniqueness constraint when the group exists.
func xmlExportWithItems(tmplName, tgName, itemKey string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<zabbix_export>
  <version>7.0</version>
  <templates>
    <template>
      <uuid>%s</uuid>
      <template>%s</template>
      <name>%s</name>
      <groups>
        <group>
          <name>%s</name>
        </group>
      </groups>
      <items>
        <item>
          <uuid>%s</uuid>
          <name>CPU utilization</name>
          <key>%s</key>
          <delay>60s</delay>
          <value_type>FLOAT</value_type>
        </item>
      </items>
    </template>
  </templates>
</zabbix_export>`, randomUUID(), tmplName, tmplName, tgName, randomUUID(), itemKey)
}

func cleanupImportedTemplate(t *testing.T, cfg *testhelper.Config, tmplName string) {
	t.Helper()
	c, err := client.New(context.Background(), cfg.URL, cfg.Token)
	if err != nil {
		t.Logf("cleanupImportedTemplate: client.New: %v", err)
		return
	}
	tmpls, err := client.TemplateGetByHost(context.Background(), c, tmplName)
	if err != nil || len(tmpls) == 0 {
		return
	}
	if err := client.TemplateDelete(context.Background(), c, tmpls[0].TemplateID); err != nil {
		t.Logf("cleanupImportedTemplate: TemplateDelete: %v", err)
	}
}
