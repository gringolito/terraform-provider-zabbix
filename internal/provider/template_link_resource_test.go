package provider_test

import (
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTemplateLinkResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	parentName := cfg.NamePrefix + "-parent"
	childName := cfg.NamePrefix + "-child"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccTemplateLinkResourceConfig(cfg, tgName, parentName, childName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template_link.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_template_link.test",
						tfjsonpath.New("on_destroy"),
						knownvalue.StringExact("clear"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccTemplateLinkResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	parentName := cfg.NamePrefix + "-parent-imp"
	childName := cfg.NamePrefix + "-child-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateLinkResourceConfig(cfg, tgName, parentName, childName),
			},
			{
				ResourceName:      "zabbix_template_link.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTemplateLinkResource_OnDestroyUnlink(t *testing.T) {
	cfg := testhelper.Setup(t)
	tgName := cfg.NamePrefix + "-tg"
	parentName := cfg.NamePrefix + "-parent-unlink"
	childName := cfg.NamePrefix + "-child-unlink"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateLinkResourceWithOnDestroyConfig(cfg, tgName, parentName, childName, "unlink"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_template_link.test",
						tfjsonpath.New("on_destroy"),
						knownvalue.StringExact("unlink"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func testAccTemplateLinkResourceConfig(cfg *testhelper.Config, tgName, parentName, childName string) string {
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
}

resource "zabbix_template_link" "test" {
  template_id        = zabbix_template.child.id
  linked_template_id = zabbix_template.parent.id
}
`, cfg.URL, cfg.Token, tgName, parentName, childName)
}

func testAccTemplateLinkResourceWithOnDestroyConfig(cfg *testhelper.Config, tgName, parentName, childName, onDestroy string) string {
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
}

resource "zabbix_template_link" "test" {
  template_id        = zabbix_template.child.id
  linked_template_id = zabbix_template.parent.id
  on_destroy         = %[6]q
}
`, cfg.URL, cfg.Token, tgName, parentName, childName, onDestroy)
}
