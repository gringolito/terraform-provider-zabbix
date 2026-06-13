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

func TestAccItemDataSource_ByID(t *testing.T) {
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
				Config: testAccItemDataSourceByIDConfig(cfg, tgName, tmplName, itemKey),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_item.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_item.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact(itemKey),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_item.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("CPU utilization"),
					),
				},
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

func TestAccItemDataSource_ByKeyAndTemplateID(t *testing.T) {
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
				Config: testAccItemDataSourceByKeyAndTemplateIDConfig(cfg, tgName, tmplName, itemKey),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_item.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_item.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact(itemKey),
					),
				},
				PostApplyFunc: func() {
					cleanupImportedTemplate(t, cfg, tmplName)
				},
			},
		},
	})
}

// ---- Unit tests ----

func TestItemDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeItemDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildItemDataSourceConfig(t, "", "", "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when no lookup key is set")
	}
}

func TestItemDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"itemid": "1", "key_": "cpu.util", "name": "CPU 1", "hostid": "42"},
			{"itemid": "2", "key_": "cpu.util", "name": "CPU 2", "hostid": "42"},
		},
	}

	ds := newFakeItemDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildItemDataSourceConfig(t, "", "cpu.util", "42", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple items found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple items found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestItemDataSource_ZeroMatchError(t *testing.T) {
	fake := &clienttest.TestClient{Response: []map[string]any{}}

	ds := newFakeItemDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildItemDataSourceConfig(t, "", "not.found", "1", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for zero matches")
	}
}

// ---- helpers ----

func newFakeItemDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewItemDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildItemDataSourceConfig(t *testing.T, id, key, hostID, templateID string) tfsdk.Config {
	t.Helper()

	null := func(ty tftypes.Type) tftypes.Value { return tftypes.NewValue(ty, nil) }
	toStr := func(s string) tftypes.Value {
		if s == "" {
			return null(tftypes.String)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewItemDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":          tftypes.String,
				"key":         tftypes.String,
				"name":        tftypes.String,
				"host_id":     tftypes.String,
				"template_id": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"id":          toStr(id),
			"key":         toStr(key),
			"name":        null(tftypes.String),
			"host_id":     toStr(hostID),
			"template_id": toStr(templateID),
		}),
		Schema: schemaResp.Schema,
	}
}

// ---- config helpers ----

func testAccItemDataSourceByIDConfig(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	return testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey) + fmt.Sprintf(`
data "zabbix_template" "seed" {
  depends_on = [zabbix_template_group.test]
  host       = %[1]q
}

data "zabbix_item" "by_key" {
  depends_on  = [zabbix_template_group.test]
  key         = %[2]q
  template_id = data.zabbix_template.seed.id
}

data "zabbix_item" "test" {
  id = data.zabbix_item.by_key.id
}
`, tmplName, itemKey)
}

func testAccItemDataSourceByKeyAndTemplateIDConfig(cfg *testhelper.Config, tgName, tmplName, itemKey string) string {
	return testAccTriggerImportSetup(cfg, tgName, tmplName, itemKey) + fmt.Sprintf(`
data "zabbix_template" "seed" {
  depends_on = [zabbix_template_group.test]
  host       = %[1]q
}

data "zabbix_item" "test" {
  depends_on  = [zabbix_template_group.test]
  key         = %[2]q
  template_id = data.zabbix_template.seed.id
}
`, tmplName, itemKey)
}
