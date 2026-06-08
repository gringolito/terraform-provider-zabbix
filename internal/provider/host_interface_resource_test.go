package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// ---- Acceptance tests ----

func TestAccHostInterfaceResource_CRUD_Agent(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccHostInterfaceResourceAgentConfig(cfg, hgName, hostName, "10050"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("agent"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("ip"),
						knownvalue.StringExact("192.168.1.1"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("port"),
						knownvalue.StringExact("10050"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("use_ip"),
						knownvalue.Bool(true),
					),
				},
			},
			// Update port
			{
				Config: testAccHostInterfaceResourceAgentConfig(cfg, hgName, hostName, "10051"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("port"),
						knownvalue.StringExact("10051"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccHostInterfaceResource_CRUD_SNMP(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-snmp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccHostInterfaceResourceSNMPConfig(cfg, hgName, hostName, "10.0.0.1", "public"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("snmp"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update community string
			{
				Config: testAccHostInterfaceResourceSNMPConfig(cfg, hgName, hostName, "10.0.0.1", "private"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_interface.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("snmp"),
					),
				},
			},
		},
	})
}

func TestAccHostInterfaceResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostInterfaceResourceAgentConfig(cfg, hgName, hostName, "10050"),
			},
			{
				ResourceName:      "zabbix_host_interface.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccHostInterfaceResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-drift"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccHostInterfaceResourceAgentConfig(cfg, hgName, hostName, "10050"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_host_interface.test"]
						if rs == nil {
							return fmt.Errorf("zabbix_host_interface.test not found in state")
						}
						capturedID = rs.Primary.ID
						return nil
					},
				),
			},
			// Delete out-of-band to trigger drift detection
			{
				PreConfig: func() {
					c, err := client.New(context.Background(), cfg.URL, cfg.Token)
					if err != nil {
						t.Fatalf("drift PreConfig: client.New: %v", err)
					}
					if err := client.HostInterfaceDelete(context.Background(), c, capturedID); err != nil {
						t.Fatalf("drift PreConfig: HostInterfaceDelete: %v", err)
					}
				},
				Config:             testAccHostInterfaceResourceAgentConfig(cfg, hgName, hostName, "10050"),
				ExpectNonEmptyPlan: true,
				// Plan should detect drift and recreate
			},
		},
	})
}

// ---- Unit tests ----

func TestHostInterfaceResource_SNMPBlockOnNonSNMPError(t *testing.T) {
	cfg := &testhelper.Config{
		URL:   "http://fake:8080",
		Token: "fake-token",
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHostInterfaceSNMPBlockOnAgentConfig(cfg),
				ExpectError: regexp.MustCompile(`snmp block.*only.*type.*snmp|type.*snmp.*snmp block`),
			},
		},
	})
}

// ---- config helpers ----

func testAccHostInterfaceResourceAgentConfig(cfg *testhelper.Config, hgName, hostName, port string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_group" "test" {
  name = %[3]q
}

resource "zabbix_host" "test" {
  host           = %[4]q
  host_group_ids = [zabbix_host_group.test.id]
}

resource "zabbix_host_interface" "test" {
  host_id = zabbix_host.test.id
  type    = "agent"
  use_ip  = true
  ip      = "192.168.1.1"
  dns     = ""
  port    = %[5]q
  main    = true
}
`, cfg.URL, cfg.Token, hgName, hostName, port)
}

func testAccHostInterfaceResourceSNMPConfig(cfg *testhelper.Config, hgName, hostName, ip, community string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_group" "test" {
  name = %[3]q
}

resource "zabbix_host" "test" {
  host           = %[4]q
  host_group_ids = [zabbix_host_group.test.id]
}

resource "zabbix_host_interface" "test" {
  host_id = zabbix_host.test.id
  type    = "snmp"
  use_ip  = true
  ip      = %[5]q
  dns     = ""
  port    = "161"
  main    = true

  snmp = {
    version   = "v2c"
    community = %[6]q
  }
}
`, cfg.URL, cfg.Token, hgName, hostName, ip, community)
}

func testAccHostInterfaceSNMPBlockOnAgentConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_interface" "test" {
  host_id = "999"
  type    = "agent"
  use_ip  = true
  ip      = "127.0.0.1"
  dns     = ""
  port    = "10050"
  main    = true

  snmp = {
    version   = "v2c"
    community = "public"
  }
}
`, cfg.URL, cfg.Token)
}
