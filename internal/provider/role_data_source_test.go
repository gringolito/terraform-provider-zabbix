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

func TestAccRoleDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-role-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_role.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccRoleDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-role-ds-name"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_role.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccRoleDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleDataSourceByNameOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`Role not found`),
			},
		},
	})
}

func TestAccRoleDataSource_BuiltinRole(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceByNameOnlyConfig(cfg, "Admin role"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Admin role"),
					),
				},
			},
		},
	})
}

// ---- Unit tests ----

func TestRoleDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{
				"roleid": "1", "name": "Custom", "type": "1", "readonly": "0",
				"rules": map[string]any{
					"ui": []any{}, "ui.default_access": "1",
					"api.access": "1", "api.mode": "0", "api.methods": []any{},
					"modules": []any{}, "actions": []any{},
				},
			},
			{
				"roleid": "2", "name": "Custom", "type": "1", "readonly": "0",
				"rules": map[string]any{
					"ui": []any{}, "ui.default_access": "1",
					"api.access": "1", "api.mode": "0", "api.methods": []any{},
					"modules": []any{}, "actions": []any{},
				},
			},
		},
	}

	ds := newFakeRoleDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildRoleDataSourceConfig(t, "", "Custom")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple roles found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple roles found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestRoleDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeRoleDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildRoleDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor name is set")
	}
}

// ---- helpers ----

func newFakeRoleDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewRoleDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildRoleDataSourceConfig(t *testing.T, id, name string) tfsdk.Config {
	t.Helper()

	toVal := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewRoleDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	rulesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"ui": tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"name":    tftypes.String,
					"enabled": tftypes.Bool,
				},
			}},
			"ui_default_access":      tftypes.Bool,
			"modules_default_access": tftypes.Bool,
			"actions_default_access": tftypes.Bool,
			"api_access":             tftypes.Bool,
			"api_mode":               tftypes.String,
			"api_methods": tftypes.Set{
				ElementType: tftypes.String,
			},
			"modules": tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"module_id": tftypes.String,
					"enabled":   tftypes.Bool,
				},
			}},
			"actions": tftypes.List{ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"name":    tftypes.String,
					"enabled": tftypes.Bool,
				},
			}},
		},
	}

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":    tftypes.String,
				"name":  tftypes.String,
				"type":  tftypes.String,
				"rules": rulesType,
			},
		}, map[string]tftypes.Value{
			"id":    toVal(id),
			"name":  toVal(name),
			"type":  tftypes.NewValue(tftypes.String, nil),
			"rules": tftypes.NewValue(rulesType, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccRoleDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_role" "seed" {
  name = %[3]q
  type = "user"
}

data "zabbix_role" "test" {
  id = zabbix_role.seed.id
}
`, cfg.URL, cfg.Token, name)
}

func testAccRoleDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_role" "seed" {
  name = %[3]q
  type = "user"
}

data "zabbix_role" "test" {
  name       = zabbix_role.seed.name
  depends_on = [zabbix_role.seed]
}
`, cfg.URL, cfg.Token, name)
}

func testAccRoleDataSourceByNameOnlyConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_role" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
