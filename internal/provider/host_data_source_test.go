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

func TestAccHostDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostDataSourceByIDConfig(cfg, hgName, hostName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_host.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(hostName),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_host.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccHostDataSource_ByTechnicalName(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-ds-name"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostDataSourceByTechnicalNameConfig(cfg, hgName, hostName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_host.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(hostName),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_host.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccHostDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHostDataSourceByTechnicalNameOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`Host not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestHostDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{
				"hostid": "1", "host": "web-srv", "name": "Web Server",
				"description": "", "status": "0",
				"groups": []map[string]any{{"groupid": "1"}},
				"tags":   []map[string]any{}, "inventory": []any{},
				"inventory_mode": "-1", "proxyid": "0",
			},
			{
				"hostid": "2", "host": "web-srv", "name": "Web Server 2",
				"description": "", "status": "0",
				"groups": []map[string]any{{"groupid": "1"}},
				"tags":   []map[string]any{}, "inventory": []any{},
				"inventory_mode": "-1", "proxyid": "0",
			},
		},
	}

	ds := newFakeHostDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildHostDataSourceConfig(t, "", "web-srv")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple hosts found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple hosts found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestHostDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeHostDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildHostDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor host is set")
	}
}

// ---- helpers ----

func newFakeHostDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewHostDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildHostDataSourceConfig(t *testing.T, id, host string) tfsdk.Config {
	t.Helper()

	null := func(ty tftypes.Type) tftypes.Value { return tftypes.NewValue(ty, nil) }
	toStr := func(s string) tftypes.Value {
		if s == "" {
			return null(tftypes.String)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	tagObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":  tftypes.String,
		"value": tftypes.String,
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewHostDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":             tftypes.String,
				"host":           tftypes.String,
				"name":           tftypes.String,
				"description":    tftypes.String,
				"status":         tftypes.String,
				"host_group_ids": tftypes.Set{ElementType: tftypes.String},
				"tags":           tftypes.Set{ElementType: tagObjType},
				"inventory_mode": tftypes.String,
				"proxy_id":       tftypes.String,
			},
		}, map[string]tftypes.Value{
			"id":             toStr(id),
			"host":           toStr(host),
			"name":           null(tftypes.String),
			"description":    null(tftypes.String),
			"status":         null(tftypes.String),
			"host_group_ids": null(tftypes.Set{ElementType: tftypes.String}),
			"tags":           null(tftypes.Set{ElementType: tagObjType}),
			"inventory_mode": null(tftypes.String),
			"proxy_id":       null(tftypes.String),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccHostDataSourceByIDConfig(cfg *testhelper.Config, hgName, hostName string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_group" "seed" {
  name = %[3]q
}

resource "zabbix_host" "seed" {
  host           = %[4]q
  host_group_ids = [zabbix_host_group.seed.id]
}

data "zabbix_host" "test" {
  id = zabbix_host.seed.id
}
`, cfg.URL, cfg.Token, hgName, hostName)
}

func testAccHostDataSourceByTechnicalNameConfig(cfg *testhelper.Config, hgName, hostName string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_group" "seed" {
  name = %[3]q
}

resource "zabbix_host" "seed" {
  host           = %[4]q
  host_group_ids = [zabbix_host_group.seed.id]
}

data "zabbix_host" "test" {
  host       = zabbix_host.seed.host
  depends_on = [zabbix_host.seed]
}
`, cfg.URL, cfg.Token, hgName, hostName)
}

func testAccHostDataSourceByTechnicalNameOnlyConfig(cfg *testhelper.Config, hostName string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_host" "test" {
  host = %[3]q
}
`, cfg.URL, cfg.Token, hostName)
}
