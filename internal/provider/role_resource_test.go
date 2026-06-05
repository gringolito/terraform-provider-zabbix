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

func TestAccRoleResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-role"
	updated := cfg.NamePrefix + "-role-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccRoleResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(initial),
					),
					statecheck.ExpectKnownValue(
						"zabbix_role.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_role.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("user"),
					),
				},
			},
			// Update name
			{
				Config: testAccRoleResourceUpdatedConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updated),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccRoleResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-role-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResourceConfig(cfg, name),
			},
			{
				ResourceName:      "zabbix_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRoleResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-role-drift"
	renamed := cfg.NamePrefix + "-role-drift-oob"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccRoleResourceConfig(cfg, name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_role.test"]
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
					role := client.Role{ID: capturedID, Name: renamed, Type: 1}
					if err := client.RoleUpdate(context.Background(), c, role); err != nil {
						t.Fatalf("drift PreConfig: rename out-of-band: %v", err)
					}
				},
				Config: testAccRoleResourceConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

func testAccRoleResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_role" "test" {
  name = %[3]q
  type = "user"
}
`, cfg.URL, cfg.Token, name)
}

func testAccRoleResourceUpdatedConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_role" "test" {
  name = %[3]q
  type = "admin"
}
`, cfg.URL, cfg.Token, name)
}
