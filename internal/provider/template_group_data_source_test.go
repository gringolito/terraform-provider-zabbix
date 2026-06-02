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

func TestAccTemplateGroupDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-tg-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateGroupDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_template_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_template_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTemplateGroupDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-tg-ds-name"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateGroupDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_template_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_template_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTemplateGroupDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTemplateGroupDataSourceByNameOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`Template group not found`),
			},
		},
	})
}

// ---- Unit tests ----

// TestTemplateGroupDataSource_MultipleMatchError verifies the guard against multiple
// results. Zabbix enforces unique names so this path is exercised via a fake client.
func TestTemplateGroupDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"groupid": "1", "name": "Templates"},
			{"groupid": "2", "name": "Templates"},
		},
	}

	ds := newFakeTemplateGroupDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildTemplateGroupDataSourceConfig(t, "", "Templates")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple template groups found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple template groups found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestTemplateGroupDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeTemplateGroupDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildTemplateGroupDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor name is set")
	}
}

// ---- helpers ----

// newFakeTemplateGroupDataSource creates a TemplateGroupDataSource with the given fake
// client injected via the framework's Configure mechanism.
func newFakeTemplateGroupDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewTemplateGroupDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildTemplateGroupDataSourceConfig(t *testing.T, id, name string) tfsdk.Config {
	t.Helper()

	toVal := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewTemplateGroupDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":   tftypes.String,
				"name": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"id":   toVal(id),
			"name": toVal(name),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccTemplateGroupDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "seed" {
  name = %[3]q
}

data "zabbix_template_group" "test" {
  id = zabbix_template_group.seed.id
}
`, cfg.URL, cfg.Token, name)
}

func testAccTemplateGroupDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "seed" {
  name = %[3]q
}

data "zabbix_template_group" "test" {
  name       = zabbix_template_group.seed.name
  depends_on = [zabbix_template_group.seed]
}
`, cfg.URL, cfg.Token, name)
}

func testAccTemplateGroupDataSourceByNameOnlyConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_template_group" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
