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

func TestAccUserGroupDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ug-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserGroupDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_user_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccUserGroupDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ug-ds-name"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserGroupDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_user_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccUserGroupDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUserGroupDataSourceByNameOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`User group not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestUserGroupDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"usrgrpid": "1", "name": "Admins", "gui_access": "0", "debug_mode": "0", "users_status": "0"},
			{"usrgrpid": "2", "name": "Admins", "gui_access": "0", "debug_mode": "0", "users_status": "0"},
		},
	}

	ds := newFakeUserGroupDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildUserGroupDataSourceConfig(t, "", "Admins")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple user groups found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple user groups found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestUserGroupDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeUserGroupDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildUserGroupDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor name is set")
	}
}

// ---- helpers ----

func newFakeUserGroupDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewUserGroupDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildUserGroupDataSourceConfig(t *testing.T, id, name string) tfsdk.Config {
	t.Helper()

	toVal := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewUserGroupDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":           tftypes.String,
				"name":         tftypes.String,
				"gui_access":   tftypes.Number,
				"debug_mode":   tftypes.Number,
				"users_status": tftypes.Number,
			},
		}, map[string]tftypes.Value{
			"id":           toVal(id),
			"name":         toVal(name),
			"gui_access":   tftypes.NewValue(tftypes.Number, nil),
			"debug_mode":   tftypes.NewValue(tftypes.Number, nil),
			"users_status": tftypes.NewValue(tftypes.Number, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccUserGroupDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_user_group" "seed" {
  name = %[3]q
}

data "zabbix_user_group" "test" {
  id = zabbix_user_group.seed.id
}
`, cfg.URL, cfg.Token, name)
}

func testAccUserGroupDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_user_group" "seed" {
  name = %[3]q
}

data "zabbix_user_group" "test" {
  name       = zabbix_user_group.seed.name
  depends_on = [zabbix_user_group.seed]
}
`, cfg.URL, cfg.Token, name)
}

func testAccUserGroupDataSourceByNameOnlyConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_user_group" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
