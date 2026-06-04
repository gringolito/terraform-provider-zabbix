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

func TestAccUserGroupResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-ug"
	updated := cfg.NamePrefix + "-ug-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccUserGroupResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(initial),
					),
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("gui_access"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("debug_mode"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("users_status"),
						knownvalue.Int64Exact(0),
					),
				},
			},
			// Update name
			{
				Config: testAccUserGroupResourceConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updated),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccUserGroupResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ug-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserGroupResourceConfig(cfg, name),
			},
			{
				ResourceName:      "zabbix_user_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccUserGroupResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ug-drift"
	renamed := cfg.NamePrefix + "-ug-drift-oob"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccUserGroupResourceConfig(cfg, name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_user_group.test"]
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
					ug := client.UserGroup{ID: capturedID, Name: renamed}
					if err := client.UserGroupUpdate(context.Background(), c, ug); err != nil {
						t.Fatalf("drift PreConfig: rename out-of-band: %v", err)
					}
				},
				Config: testAccUserGroupResourceConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_user_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

func testAccUserGroupResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_user_group" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
