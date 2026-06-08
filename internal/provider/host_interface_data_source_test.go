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

func TestAccHostInterfaceDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostInterfaceDataSourceByIDConfig(cfg, hgName, hostName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_host_interface.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_host_interface.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("agent"),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_host_interface.test",
						tfjsonpath.New("port"),
						knownvalue.StringExact("10050"),
					),
				},
			},
		},
	})
}

func TestAccHostInterfaceDataSource_ByHostAndType(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-ds-htype"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostInterfaceDataSourceByHostAndTypeConfig(cfg, hgName, hostName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.zabbix_host_interface.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.zabbix_host_interface.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("agent"),
					),
				},
			},
		},
	})
}

func TestAccHostInterfaceDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHostInterfaceDataSourceByIDOnlyConfig(cfg, "99999999"),
				ExpectError: regexp.MustCompile(`Host interface not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestHostInterfaceDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{
				"interfaceid": "1", "hostid": "10084", "type": "1",
				"useip": "1", "ip": "127.0.0.1", "dns": "", "port": "10050",
				"main": "1", "details": []any{},
			},
			{
				"interfaceid": "2", "hostid": "10084", "type": "1",
				"useip": "1", "ip": "10.0.0.1", "dns": "", "port": "10050",
				"main": "0", "details": []any{},
			},
		},
	}

	ds := newFakeHostInterfaceDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildHostInterfaceDataSourceConfig(t, "", "10084", "agent")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple host interfaces found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple host interfaces found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestHostInterfaceDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeHostInterfaceDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildHostInterfaceDataSourceConfig(t, "", "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when no lookup key is set")
	}
}

func TestHostInterfaceDataSource_ZeroMatchByHostAndTypeError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{},
	}

	ds := newFakeHostInterfaceDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildHostInterfaceDataSourceConfig(t, "", "10084", "agent")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for zero matches")
	}
}

// ---- helpers ----

func newFakeHostInterfaceDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewHostInterfaceDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildHostInterfaceDataSourceConfig(t *testing.T, id, hostID, ifaceType string) tfsdk.Config {
	t.Helper()

	null := func(ty tftypes.Type) tftypes.Value { return tftypes.NewValue(ty, nil) }
	toStr := func(s string) tftypes.Value {
		if s == "" {
			return null(tftypes.String)
		}
		return tftypes.NewValue(tftypes.String, s)
	}

	snmpObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"version":         tftypes.String,
		"community":       tftypes.String,
		"bulk_requests":   tftypes.Bool,
		"security_name":   tftypes.String,
		"security_level":  tftypes.String,
		"auth_protocol":   tftypes.String,
		"auth_passphrase": tftypes.String,
		"priv_protocol":   tftypes.String,
		"priv_passphrase": tftypes.String,
		"context_name":    tftypes.String,
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewHostInterfaceDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":      tftypes.String,
				"host_id": tftypes.String,
				"type":    tftypes.String,
				"use_ip":  tftypes.Bool,
				"ip":      tftypes.String,
				"dns":     tftypes.String,
				"port":    tftypes.String,
				"main":    tftypes.Bool,
				"snmp":    snmpObjType,
			},
		}, map[string]tftypes.Value{
			"id":      toStr(id),
			"host_id": toStr(hostID),
			"type":    toStr(ifaceType),
			"use_ip":  null(tftypes.Bool),
			"ip":      null(tftypes.String),
			"dns":     null(tftypes.String),
			"port":    null(tftypes.String),
			"main":    null(tftypes.Bool),
			"snmp":    null(snmpObjType),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccHostInterfaceDataSourceByIDConfig(cfg *testhelper.Config, hgName, hostName string) string {
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

resource "zabbix_host_interface" "seed" {
  host_id = zabbix_host.seed.id
  type    = "agent"
  use_ip  = true
  ip      = "127.0.0.1"
  dns     = ""
  port    = "10050"
  main    = true
}

data "zabbix_host_interface" "test" {
  id = zabbix_host_interface.seed.id
}
`, cfg.URL, cfg.Token, hgName, hostName)
}

func testAccHostInterfaceDataSourceByHostAndTypeConfig(cfg *testhelper.Config, hgName, hostName string) string {
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

resource "zabbix_host_interface" "seed" {
  host_id = zabbix_host.seed.id
  type    = "agent"
  use_ip  = true
  ip      = "127.0.0.1"
  dns     = ""
  port    = "10050"
  main    = true
}

data "zabbix_host_interface" "test" {
  host_id    = zabbix_host.seed.id
  type       = "agent"
  depends_on = [zabbix_host_interface.seed]
}
`, cfg.URL, cfg.Token, hgName, hostName)
}

func testAccHostInterfaceDataSourceByIDOnlyConfig(cfg *testhelper.Config, id string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_host_interface" "test" {
  id = %[3]q
}
`, cfg.URL, cfg.Token, id)
}
