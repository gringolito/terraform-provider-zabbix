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
)

// ---- Acceptance tests ----

func TestAccTriggerDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerDataSourceByIDConfig(cfg, tgName, tmplName, itemKey),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_trigger.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("High CPU"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_trigger.test",
						tfjsonpath.New("priority"),
						knownvalue.StringExact("warning"),
					),
				},
			},
		},
	})
}

func TestAccTriggerDataSource_ByDescriptionAndTemplateID(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	tmplName := cfg.NamePrefix + "-tmpl"
	itemKey := "system.cpu.util"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerDataSourceByDescriptionAndTemplateIDConfig(cfg, tgName, tmplName, itemKey),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_trigger.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_trigger.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("High CPU"),
					),
				},
			},
		},
	})
}

// ---- Unit tests ----

func TestTriggerDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeTriggerDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildTriggerDataSourceConfig(t, "", "", "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when no lookup key is set")
	}
}

func TestTriggerDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{
				"triggerid": "1", "description": "High CPU",
				"expression": "last(/h/k)>90", "recovery_mode": "0",
				"recovery_expression": "", "priority": "2", "status": "0",
				"manual_close": "0", "comments": "", "url": "", "tags": []any{},
			},
			{
				"triggerid": "2", "description": "High CPU",
				"expression": "last(/h2/k)>90", "recovery_mode": "0",
				"recovery_expression": "", "priority": "2", "status": "0",
				"manual_close": "0", "comments": "", "url": "", "tags": []any{},
			},
		},
	}

	ds := newFakeTriggerDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildTriggerDataSourceConfig(t, "", "High CPU", "42", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple triggers found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple triggers found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestTriggerDataSource_ZeroMatchError(t *testing.T) {
	fake := &clienttest.TestClient{Response: []map[string]any{}}

	ds := newFakeTriggerDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildTriggerDataSourceConfig(t, "", "Not found", "1", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for zero matches")
	}
}

// ---- helpers ----

func newFakeTriggerDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewTriggerDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildTriggerDataSourceConfig(t *testing.T, id, description, hostID, templateID string) tfsdk.Config {
	t.Helper()

	null := func(ty tftypes.Type) tftypes.Value { return tftypes.NewValue(ty, nil) }
	toStr := func(s string) tftypes.Value {
		if s == "" {
			return null(tftypes.String)
		}
		return tftypes.NewValue(tftypes.String, s)
	}
	tagType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":  tftypes.String,
		"value": tftypes.String,
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewTriggerDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":                  tftypes.String,
				"description":         tftypes.String,
				"host_id":             tftypes.String,
				"template_id":         tftypes.String,
				"expression":          tftypes.String,
				"recovery_mode":       tftypes.String,
				"recovery_expression": tftypes.String,
				"priority":            tftypes.String,
				"status":              tftypes.String,
				"manual_close":        tftypes.Bool,
				"comments":            tftypes.String,
				"url":                 tftypes.String,
				"tags":                tftypes.Set{ElementType: tagType},
			},
		}, map[string]tftypes.Value{
			"id":                  toStr(id),
			"description":         toStr(description),
			"host_id":             toStr(hostID),
			"template_id":         toStr(templateID),
			"expression":          null(tftypes.String),
			"recovery_mode":       null(tftypes.String),
			"recovery_expression": null(tftypes.String),
			"priority":            null(tftypes.String),
			"status":              null(tftypes.String),
			"manual_close":        null(tftypes.Bool),
			"comments":            null(tftypes.String),
			"url":                 null(tftypes.String),
			"tags":                tftypes.NewValue(tftypes.Set{ElementType: tagType}, []tftypes.Value{}),
		}),
		Schema: schemaResp.Schema,
	}
}

// ---- config helpers ----

func testAccTriggerDataSourceByIDConfig(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	expr := fmt.Sprintf(`last(/%s/%s)>90`, tmplName, itemKey)
	return testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey) + fmt.Sprintf(`
resource "zabbix_trigger" "seed" {
  depends_on  = [zabbix_template_import.test]
  description = "High CPU"
  expression  = %[1]q
  priority    = "warning"
}

data "zabbix_trigger" "test" {
  id = zabbix_trigger.seed.id
}
`, expr)
}

func testAccTriggerDataSourceByDescriptionAndTemplateIDConfig(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	expr := fmt.Sprintf(`last(/%s/%s)>90`, tmplName, itemKey)
	return testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey) + fmt.Sprintf(`
resource "zabbix_template" "seed" {
  depends_on          = [zabbix_template_import.test]
  host                = %[2]q
  template_group_ids  = [zabbix_template_group.test.id]
}

resource "zabbix_trigger" "seed" {
  depends_on  = [zabbix_template.seed]
  description = "High CPU"
  expression  = %[1]q
  priority    = "warning"
}

data "zabbix_trigger" "test" {
  depends_on   = [zabbix_trigger.seed]
  description  = "High CPU"
  template_id  = zabbix_template.seed.id
}
`, expr, tmplName)
}
