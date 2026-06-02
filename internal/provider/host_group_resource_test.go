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

func TestAccHostGroupResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-hg"
	updated := cfg.NamePrefix + "-hg-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccHostGroupResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(initial),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update
			{
				Config: testAccHostGroupResourceConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updated),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccHostGroupResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-hg-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostGroupResourceConfig(cfg, name),
			},
			{
				ResourceName:      "zabbix_host_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccHostGroupResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-hg-drift"
	renamed := cfg.NamePrefix + "-hg-drift-oob"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccHostGroupResourceConfig(cfg, name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_host_group.test"]
						if rs == nil {
							return fmt.Errorf("resource not found in state")
						}
						capturedID = rs.Primary.ID
						return nil
					},
				),
			},
			// Rename out-of-band, then verify apply reconciles drift
			{
				PreConfig: func() {
					c, err := client.New(context.Background(), cfg.URL, cfg.Token)
					if err != nil {
						t.Fatalf("drift PreConfig: client.New: %v", err)
					}
					if err := client.HostGroupUpdate(context.Background(), c, capturedID, renamed); err != nil {
						t.Fatalf("drift PreConfig: rename out-of-band: %v", err)
					}
				},
				Config: testAccHostGroupResourceConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

func testAccHostGroupResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_host_group" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
