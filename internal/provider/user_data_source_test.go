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

func TestAccUserDataSource_ByUsername(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceByUsernameConfig(cfg, "Admin"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("Admin"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("super_admin"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("gui_access"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("debug_mode"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("users_status"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("language"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("timezone"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("rows_per_page"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccUserDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceByUsernameConfig(cfg, "Admin"),
			},
			{
				Config: testAccUserDataSourceByIDConfig(cfg),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("Admin"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_user.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("super_admin"),
					),
				},
			},
		},
	})
}

func TestAccUserDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUserDataSourceByUsernameConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`User not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestUserDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeUserDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildUserDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor username is set")
	}
}

func TestUserDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			testFakeUserResponse("1", "Admin"),
			testFakeUserResponse("2", "Admin"),
		},
	}

	ds := newFakeUserDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildUserDataSourceConfig(t, "", "Admin")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple users found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple users found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestUserDataSource_ZeroMatchError(t *testing.T) {
	fake := &clienttest.TestClient{Response: []map[string]any{}}

	ds := newFakeUserDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildUserDataSourceConfig(t, "", "nonexistent")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for zero matches")
	}
}

// ---- helpers ----

func testFakeUserResponse(id, username string) map[string]any {
	return map[string]any{
		"userid":         id,
		"username":       username,
		"name":           "Zabbix",
		"surname":        "Administrator",
		"url":            "",
		"autologin":      "0",
		"autologout":     "0",
		"lang":           "default",
		"refresh":        "30s",
		"theme":          "default",
		"attempt_failed": "0",
		"attempt_ip":     "",
		"attempt_clock":  "0",
		"rows_per_page":  "50",
		"timezone":       "default",
		"roleid":         "3",
		"provisioned":    "0",
		"gui_access":     "0",
		"debug_mode":     "0",
		"users_status":   "0",
		"type":           "3",
	}
}

func newFakeUserDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewUserDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildUserDataSourceConfig(t *testing.T, id, username string) tfsdk.Config {
	t.Helper()

	toVal := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewUserDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":             tftypes.String,
				"username":       tftypes.String,
				"name":           tftypes.String,
				"surname":        tftypes.String,
				"url":            tftypes.String,
				"auto_login":     tftypes.String,
				"auto_logout":    tftypes.String,
				"language":       tftypes.String,
				"refresh":        tftypes.String,
				"theme":          tftypes.String,
				"attempt_failed": tftypes.String,
				"attempt_ip":     tftypes.String,
				"attempt_clock":  tftypes.String,
				"rows_per_page":  tftypes.String,
				"timezone":       tftypes.String,
				"provisioned":    tftypes.String,
				"gui_access":     tftypes.String,
				"debug_mode":     tftypes.String,
				"users_status":   tftypes.String,
				"type":           tftypes.String,
				"role_id":        tftypes.String,
			},
		}, map[string]tftypes.Value{
			"id":             toVal(id),
			"username":       toVal(username),
			"name":           tftypes.NewValue(tftypes.String, nil),
			"surname":        tftypes.NewValue(tftypes.String, nil),
			"url":            tftypes.NewValue(tftypes.String, nil),
			"auto_login":     tftypes.NewValue(tftypes.String, nil),
			"auto_logout":    tftypes.NewValue(tftypes.String, nil),
			"language":       tftypes.NewValue(tftypes.String, nil),
			"refresh":        tftypes.NewValue(tftypes.String, nil),
			"theme":          tftypes.NewValue(tftypes.String, nil),
			"attempt_failed": tftypes.NewValue(tftypes.String, nil),
			"attempt_ip":     tftypes.NewValue(tftypes.String, nil),
			"attempt_clock":  tftypes.NewValue(tftypes.String, nil),
			"rows_per_page":  tftypes.NewValue(tftypes.String, nil),
			"timezone":       tftypes.NewValue(tftypes.String, nil),
			"provisioned":    tftypes.NewValue(tftypes.String, nil),
			"gui_access":     tftypes.NewValue(tftypes.String, nil),
			"debug_mode":     tftypes.NewValue(tftypes.String, nil),
			"users_status":   tftypes.NewValue(tftypes.String, nil),
			"type":           tftypes.NewValue(tftypes.String, nil),
			"role_id":        tftypes.NewValue(tftypes.String, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

// ---- config helpers ----

func testAccUserDataSourceByUsernameConfig(cfg *testhelper.Config, username string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_user" "test" {
  username = %[3]q
}
`, cfg.URL, cfg.Token, username)
}

func testAccUserDataSourceByIDConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_user" "seed" {
  username = "Admin"
}

data "zabbix_user" "test" {
  id = data.zabbix_user.seed.id
}
`, cfg.URL, cfg.Token)
}
