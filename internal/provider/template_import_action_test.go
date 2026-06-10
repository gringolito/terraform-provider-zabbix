package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/gringolito/terraform-provider-zabbix/internal/clienttest"
	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// ---- Unit tests ----

func TestTemplateImportAction_Configure_NilData(t *testing.T) {
	a := provider.NewTemplateImportAction()
	configurable, ok := a.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("action does not implement ActionWithConfigure")
	}
	resp := &action.ConfigureResponse{}
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil provider data: %s", resp.Diagnostics)
	}
}

func TestTemplateImportAction_Configure_WrongType(t *testing.T) {
	a := provider.NewTemplateImportAction()
	configurable, ok := a.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("action does not implement ActionWithConfigure")
	}
	resp := &action.ConfigureResponse{}
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: "not-a-client"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type, got none")
	}
}

func TestTemplateImportAction_Invoke_ClientError(t *testing.T) {
	a := configuredAction(t, &clienttest.TestClient{Error: errors.New("api unavailable")})
	resp := invokeAction(t, a, "content", "xml")
	if !resp.Diagnostics.HasError() {
		t.Error("expected diagnostic error when client fails, got none")
	}
}

func TestTemplateImportAction_Invoke_Success_DefaultRules(t *testing.T) {
	mc := &clienttest.TestClient{Response: true}
	a := configuredAction(t, mc)
	resp := invokeAction(t, a, "<zabbix_export/>", "xml")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics)
	}
	if mc.LastMethod != "configuration.import" {
		t.Errorf("LastMethod = %q, want %q", mc.LastMethod, "configuration.import")
	}
	rules := importedRules(t, mc)
	if !rules.Templates.CreateMissing || !rules.Templates.UpdateExisting {
		t.Errorf("default templates rules: got %+v, want create=true update=true", rules.Templates)
	}
	if rules.TemplateLinkage.DeleteMissing {
		t.Errorf("default template_linkage delete_missing: got true, want false")
	}
}

func TestTemplateImportAction_Invoke_FormatValidation(t *testing.T) {
	mc := &clienttest.TestClient{Response: true}
	a := configuredAction(t, mc)
	resp := invokeAction(t, a, "content", "yaml")
	if resp.Diagnostics.HasError() {
		t.Fatalf("yaml format should be valid: %s", resp.Diagnostics)
	}
	if mc.LastMethod != "configuration.import" {
		t.Errorf("LastMethod = %q, want %q", mc.LastMethod, "configuration.import")
	}
}

// configuredAction wires mc into a fresh TemplateImportAction via Configure.
func configuredAction(t *testing.T, mc *clienttest.TestClient) action.Action {
	t.Helper()
	a := provider.NewTemplateImportAction()
	configurable, ok := a.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("action does not implement ActionWithConfigure")
	}
	cfgResp := &action.ConfigureResponse{}
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: mc}, cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %s", cfgResp.Diagnostics)
	}
	return a
}

// invokeAction calls Invoke with source and format, rules null.
func invokeAction(t *testing.T, a action.Action, source, format string) *action.InvokeResponse {
	t.Helper()
	resp := &action.InvokeResponse{SendProgress: func(action.InvokeProgressEvent) {}}
	a.Invoke(context.Background(), action.InvokeRequest{Config: makeTemplateImportConfig(t, source, format)}, resp)
	return resp
}

// importedRules decodes the rules sent to the mock client's last Call.
func importedRules(t *testing.T, mc *clienttest.TestClient) client.ImportRules {
	t.Helper()
	raw, err := json.Marshal(mc.LastParams)
	if err != nil {
		t.Fatalf("marshal LastParams: %v", err)
	}
	var params struct {
		Rules client.ImportRules `json:"rules"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	return params.Rules
}

// makeTemplateImportConfig builds a tfsdk.Config for the action with the given source and format,
// and rules left null (so provider defaults apply).
func makeTemplateImportConfig(t *testing.T, source, format string) tfsdk.Config {
	t.Helper()
	a := provider.NewTemplateImportAction()
	schemaResp := &action.SchemaResponse{}
	a.Schema(context.Background(), action.SchemaRequest{}, schemaResp)

	createUpdateType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"create_missing": tftypes.Bool, "update_existing": tftypes.Bool,
	}}
	createDeleteType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"create_missing": tftypes.Bool, "delete_missing": tftypes.Bool,
	}}
	allType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"create_missing": tftypes.Bool, "update_existing": tftypes.Bool, "delete_missing": tftypes.Bool,
	}}
	rulesType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"templates":           createUpdateType,
		"template_groups":     createUpdateType,
		"template_linkage":    createDeleteType,
		"discovery_rules":     allType,
		"graphs":              allType,
		"http_tests":          allType,
		"items":               allType,
		"template_dashboards": allType,
		"triggers":            allType,
		"value_maps":          allType,
	}}
	configType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"source": tftypes.String,
		"format": tftypes.String,
		"rules":  rulesType,
	}}
	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(configType, map[string]tftypes.Value{
			"source": tftypes.NewValue(tftypes.String, source),
			"format": tftypes.NewValue(tftypes.String, format),
			"rules":  tftypes.NewValue(rulesType, nil),
		}),
	}
}

// ---- Acceptance tests ----

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
