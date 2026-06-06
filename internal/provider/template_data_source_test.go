package provider_test

import (
	"context"
	"fmt"
	"regexp"
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

func TestAccTemplateDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	name := cfg.NamePrefix + "-tmpl-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateDataSourceByIDConfig(cfg, tgName, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_template.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_template.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTemplateDataSource_ByHost(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	name := cfg.NamePrefix + "-tmpl-ds-host"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateDataSourceByHostConfig(cfg, tgName, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_template.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_template.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTemplateDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTemplateDataSourceByHostOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`Template not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestTemplateDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"templateid": "1", "host": "Linux", "name": "Linux", "description": "", "groups": []map[string]any{}, "macros": []map[string]any{}, "parentTemplates": []map[string]any{}},
			{"templateid": "2", "host": "Linux", "name": "Linux", "description": "", "groups": []map[string]any{}, "macros": []map[string]any{}, "parentTemplates": []map[string]any{}},
		},
	}

	ds := newFakeTemplateDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildTemplateDataSourceConfig(t, "", "Linux")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple templates found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple templates found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestTemplateDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeTemplateDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildTemplateDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor host is set")
	}
}

// ---- helpers ----

func newFakeTemplateDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewTemplateDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildTemplateDataSourceConfig(t *testing.T, id, host string) tfsdk.Config {
	t.Helper()

	toVal := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewTemplateDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	attrTypes := map[string]tftypes.Type{
		"id":                  tftypes.String,
		"host":                tftypes.String,
		"name":                tftypes.String,
		"description":         tftypes.String,
		"template_group_ids":  tftypes.Set{ElementType: tftypes.String},
		"macros":              tftypes.Map{ElementType: tftypes.String},
		"linked_template_ids": tftypes.Set{ElementType: tftypes.String},
	}

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, map[string]tftypes.Value{
			"id":                  toVal(id),
			"host":                toVal(host),
			"name":                tftypes.NewValue(tftypes.String, nil),
			"description":         tftypes.NewValue(tftypes.String, nil),
			"template_group_ids":  tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
			"macros":              tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
			"linked_template_ids": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccTemplateDataSourceByIDConfig(cfg *testhelper.Config, tgName, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "seed" {
  name = %[3]q
}

resource "zabbix_template" "seed" {
  host               = %[4]q
  template_group_ids = [zabbix_template_group.seed.id]
}

data "zabbix_template" "test" {
  id = zabbix_template.seed.id
}
`, cfg.URL, cfg.Token, tgName, name)
}

func testAccTemplateDataSourceByHostConfig(cfg *testhelper.Config, tgName, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "seed" {
  name = %[3]q
}

resource "zabbix_template" "seed" {
  host               = %[4]q
  template_group_ids = [zabbix_template_group.seed.id]
}

data "zabbix_template" "test" {
  host       = zabbix_template.seed.host
  depends_on = [zabbix_template.seed]
}
`, cfg.URL, cfg.Token, tgName, name)
}

func testAccTemplateDataSourceByHostOnlyConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_template" "test" {
  host = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
