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

func TestAccTemplateResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	initial := cfg.NamePrefix + "-tmpl"
	updated := cfg.NamePrefix + "-tmpl-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccTemplateResourceConfig(cfg, tgName, initial, initial, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(initial),
					),
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update host name and description
			{
				Config: testAccTemplateResourceConfig(cfg, tgName, updated, updated, "A description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(updated),
					),
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A description"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccTemplateResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	name := cfg.NamePrefix + "-tmpl-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateResourceConfig(cfg, tgName, name, name, ""),
			},
			{
				ResourceName:      "zabbix_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTemplateResource_Macros(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	name := cfg.NamePrefix + "-tmpl-macro"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with macros
			{
				Config: testAccTemplateResourceWithMacrosConfig(cfg, tgName, name, map[string]string{
					"{$PORT}":    "161",
					"{$TIMEOUT}": "5",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("macros").AtMapKey("{$PORT}"),
						knownvalue.StringExact("161"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("macros").AtMapKey("{$TIMEOUT}"),
						knownvalue.StringExact("5"),
					),
				},
			},
			// Update macros (mutation drift)
			{
				Config: testAccTemplateResourceWithMacrosConfig(cfg, tgName, name, map[string]string{
					"{$PORT}": "162",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("macros").AtMapKey("{$PORT}"),
						knownvalue.StringExact("162"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("macros"),
						knownvalue.MapSizeExact(1),
					),
				},
			},
		},
	})
}

func TestAccTemplateResource_LinkedTemplates(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	parentName := cfg.NamePrefix + "-parent-tmpl"
	childName := cfg.NamePrefix + "-child-tmpl"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create child linking to parent
			{
				Config: testAccTemplateResourceWithLinkedConfig(cfg, tgName, parentName, childName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.child",
						tfjsonpath.New("linked_template_ids"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
			// Unlink template by updating the same resource (removing linked_template_ids)
			{
				Config: testAccTemplateResourceWithUnlinkedConfig(cfg, tgName, parentName, childName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.child",
						tfjsonpath.New("linked_template_ids"),
						knownvalue.SetSizeExact(0),
					),
				},
			},
		},
	})
}

func TestAccTemplateResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	name := cfg.NamePrefix + "-tmpl-drift"
	renamed := cfg.NamePrefix + "-tmpl-drift-oob"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccTemplateResourceConfig(cfg, tgName, name, name, ""),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_template.test"]
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
					tmpl, err := client.TemplateGet(context.Background(), c, capturedID)
					if err != nil || tmpl == nil {
						t.Fatalf("drift PreConfig: TemplateGet: %v", err)
					}
					tmpl.Host = renamed
					tmpl.Name = renamed
					if err := client.TemplateUpdate(context.Background(), c, *tmpl); err != nil {
						t.Fatalf("drift PreConfig: rename out-of-band: %v", err)
					}
				},
				Config: testAccTemplateResourceConfig(cfg, tgName, name, name, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template.test",
						tfjsonpath.New("host"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

func testAccTemplateResourceConfig(cfg *testhelper.Config, tgName, host, name, description string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "test" {
  name = %[3]q
}

resource "zabbix_template" "test" {
  host                = %[4]q
  name                = %[5]q
  description         = %[6]q
  template_group_ids  = [zabbix_template_group.test.id]
}
`, cfg.URL, cfg.Token, tgName, host, name, description)
}

func testAccTemplateResourceWithMacrosConfig(cfg *testhelper.Config, tgName, host string, macros map[string]string) string {
	macroLines := ""
	for k, v := range macros {
		macroLines += fmt.Sprintf("    %q = %q\n", k, v)
	}
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "test" {
  name = %[3]q
}

resource "zabbix_template" "test" {
  host               = %[4]q
  template_group_ids = [zabbix_template_group.test.id]
  macros = {
%[5]s  }
}
`, cfg.URL, cfg.Token, tgName, host, macroLines)
}

func testAccTemplateResourceWithUnlinkedConfig(cfg *testhelper.Config, tgName, parentHost, childHost string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "test" {
  name = %[3]q
}

resource "zabbix_template" "parent" {
  host               = %[4]q
  template_group_ids = [zabbix_template_group.test.id]
}

resource "zabbix_template" "child" {
  host               = %[5]q
  template_group_ids = [zabbix_template_group.test.id]
  linked_template_ids = []
}
`, cfg.URL, cfg.Token, tgName, parentHost, childHost)
}

func testAccTemplateResourceWithLinkedConfig(cfg *testhelper.Config, tgName, parentHost, childHost string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_template_group" "test" {
  name = %[3]q
}

resource "zabbix_template" "parent" {
  host               = %[4]q
  template_group_ids = [zabbix_template_group.test.id]
}

resource "zabbix_template" "child" {
  host                 = %[5]q
  template_group_ids   = [zabbix_template_group.test.id]
  linked_template_ids  = [zabbix_template.parent.id]
}
`, cfg.URL, cfg.Token, tgName, parentHost, childHost)
}
