package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccHostResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host"
	updatedHostName := cfg.NamePrefix + "-host-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccHostResourceConfig(cfg, hgName, hostName, "Test Host", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(hostName),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Host"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("enabled"),
					),
				},
			},
			// Update technical name and description
			{
				Config: testAccHostResourceConfig(cfg, hgName, updatedHostName, "Updated Host", "A description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(updatedHostName),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated Host"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A description"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccHostResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostResourceConfig(cfg, hgName, hostName, hostName, ""),
			},
			{
				ResourceName:      "zabbix_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccHostResource_Drift_HostGroupOOB(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hgOOBName := cfg.NamePrefix + "-hg-oob"
	hostName := cfg.NamePrefix + "-host-drift"

	var capturedHostID string
	var capturedExtraGroupID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create host with one group
			{
				Config: testAccHostResourceWithExtraGroupConfig(cfg, hgName, hgOOBName, hostName),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						hostRS := s.RootModule().Resources["zabbix_host.test"]
						if hostRS == nil {
							return fmt.Errorf("zabbix_host.test not found in state")
						}
						capturedHostID = hostRS.Primary.ID
						extraRS := s.RootModule().Resources["zabbix_host_group.extra"]
						if extraRS == nil {
							return fmt.Errorf("zabbix_host_group.extra not found in state")
						}
						capturedExtraGroupID = extraRS.Primary.ID
						return nil
					},
				),
			},
			// Add second group out-of-band, verify apply reconciles it away
			{
				PreConfig: func() {
					c, err := client.New(context.Background(), cfg.URL, cfg.Token)
					if err != nil {
						t.Fatalf("drift PreConfig: client.New: %v", err)
					}
					h, err := client.HostGet(context.Background(), c, capturedHostID)
					if err != nil || h == nil {
						t.Fatalf("drift PreConfig: HostGet: %v", err)
					}
					// Add the extra group out-of-band
					h.Groups = append(h.Groups, client.HostGroupRef{GroupID: capturedExtraGroupID})
					if err := client.HostUpdate(context.Background(), c, *h); err != nil {
						t.Fatalf("drift PreConfig: HostUpdate: %v", err)
					}
				},
				Config: testAccHostResourceWithExtraGroupConfig(cfg, hgName, hgOOBName, hostName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host.test",
						tfjsonpath.New("host_group_ids"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

func testAccHostResourceConfig(cfg *testhelper.Config, hgName, hostName, visibleName, description string) string {
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
  name           = %[5]q
  description    = %[6]q
  host_group_ids = [zabbix_host_group.test.id]
}
`, cfg.URL, cfg.Token, hgName, hostName, visibleName, description)
}

func testAccHostResourceWithExtraGroupConfig(cfg *testhelper.Config, hgName, hgExtraName, hostName string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_group" "test" {
  name = %[3]q
}

resource "zabbix_host_group" "extra" {
  name = %[4]q
}

resource "zabbix_host" "test" {
  host           = %[5]q
  host_group_ids = [zabbix_host_group.test.id]
}
`, cfg.URL, cfg.Token, hgName, hgExtraName, hostName)
}
