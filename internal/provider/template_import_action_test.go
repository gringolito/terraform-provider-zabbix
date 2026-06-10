package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccTemplateImportAction_XML(t *testing.T) {
	cfg := testhelper.Setup(t)
	tmplName := cfg.NamePrefix + "-xml"
	tgName := cfg.NamePrefix + "-tg"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateImportActionConfig(cfg, tgName, tmplName, "xml", xmlExport(tmplName, tgName)),
				PostApplyFunc: func() {
					verifyAndCleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTemplateImportAction_YAML(t *testing.T) {
	cfg := testhelper.Setup(t)
	tmplName := cfg.NamePrefix + "-yaml"
	tgName := cfg.NamePrefix + "-tg"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateImportActionConfig(cfg, tgName, tmplName, "yaml", yamlExport(tmplName, tgName)),
				PostApplyFunc: func() {
					verifyAndCleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTemplateImportAction_JSON(t *testing.T) {
	cfg := testhelper.Setup(t)
	tmplName := cfg.NamePrefix + "-json"
	tgName := cfg.NamePrefix + "-tg"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateImportActionConfig(cfg, tgName, tmplName, "json", jsonExport(tmplName, tgName)),
				PostApplyFunc: func() {
					verifyAndCleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccTemplateImportAction_AfterCreate(t *testing.T) {
	cfg := testhelper.Setup(t)
	tmplName := cfg.NamePrefix + "-trig"
	tgName := cfg.NamePrefix + "-tg"

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateImportActionAfterCreateConfig(cfg, tgName, tmplName),
				PostApplyFunc: func() {
					verifyAndCleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

// verifyAndCleanupImportedTemplate checks the template was imported and deletes it via the API.
func verifyAndCleanupImportedTemplate(t *testing.T, cfg *testhelper.Config, tmplName string) {
	t.Helper()
	c, err := client.New(context.Background(), cfg.URL, cfg.Token)
	if err != nil {
		t.Fatalf("verifyAndCleanup: client.New: %v", err)
	}
	tmpls, err := client.TemplateGetByHost(context.Background(), c, tmplName)
	if err != nil {
		t.Fatalf("verifyAndCleanup: TemplateGetByHost: %v", err)
	}
	if len(tmpls) == 0 {
		t.Errorf("template %q was not imported into Zabbix", tmplName)
		return
	}
	if err := client.TemplateDelete(context.Background(), c, tmpls[0].TemplateID); err != nil {
		t.Logf("verifyAndCleanup: TemplateDelete warning: %v", err)
	}
}

func testAccTemplateImportActionConfig(cfg *testhelper.Config, tgName, tmplName, format, source string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "test" {
  name = %[3]q
}

resource "terraform_data" "trigger" {
  input = %[4]q

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.zabbix_template_import.test]
    }
  }
}

action "zabbix_template_import" "test" {
  config {
    source = %[5]q
    format = %[6]q

    rules {
      templates       = { create_missing = true, update_existing = true }
      template_groups = { create_missing = true, update_existing = true }
    }
  }
}
`, cfg.URL, cfg.Token, tgName, tmplName, source, format)
}

func testAccTemplateImportActionAfterCreateConfig(cfg *testhelper.Config, tgName, tmplName string) string {
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
      actions = [action.zabbix_template_import.on_tg_create]
    }
  }
}

action "zabbix_template_import" "on_tg_create" {
  config {
    source = %[4]q
    format = "xml"
  }
}
`, cfg.URL, cfg.Token, tgName, xmlExport(tmplName, tgName))
}

// xmlExport returns a minimal Zabbix 7.0 XML export containing one template.
func xmlExport(tmplName, tgName string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<zabbix_export>
  <version>7.0</version>
  <template_groups>
    <template_group>
      <uuid>aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa</uuid>
      <name>%s</name>
    </template_group>
  </template_groups>
  <templates>
    <template>
      <uuid>bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb</uuid>
      <template>%s</template>
      <name>%s</name>
      <groups>
        <group>
          <name>%s</name>
        </group>
      </groups>
    </template>
  </templates>
</zabbix_export>`, tgName, tmplName, tmplName, tgName)
}

// yamlExport returns a minimal Zabbix 7.0 YAML export containing one template.
func yamlExport(tmplName, tgName string) string {
	return fmt.Sprintf(`zabbix_export:
  version: '7.0'
  template_groups:
    - uuid: cccccccccccccccccccccccccccccccc
      name: %s
  templates:
    - uuid: dddddddddddddddddddddddddddddddd
      template: %s
      name: %s
      groups:
        - name: %s
`, tgName, tmplName, tmplName, tgName)
}

// jsonExport returns a minimal Zabbix 7.0 JSON export containing one template.
func jsonExport(tmplName, tgName string) string {
	return fmt.Sprintf(`{
  "zabbix_export": {
    "version": "7.0",
    "template_groups": [
      {"uuid": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "name": %q}
    ],
    "templates": [
      {
        "uuid": "ffffffffffffffffffffffffffffffff",
        "template": %q,
        "name": %q,
        "groups": [{"name": %q}]
      }
    ]
  }
}`, tgName, tmplName, tmplName, tgName)
}
