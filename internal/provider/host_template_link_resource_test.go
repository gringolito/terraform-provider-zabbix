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

func TestAccHostTemplateLinkResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host"
	tgName := cfg.NamePrefix + "-tg"
	templateName := cfg.NamePrefix + "-tmpl"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostTemplateLinkResourceConfig(cfg, hgName, hostName, tgName, templateName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_template_link.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"zabbix_host_template_link.test",
						tfjsonpath.New("on_destroy"),
						knownvalue.StringExact("clear"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func TestAccHostTemplateLinkResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-imp"
	tgName := cfg.NamePrefix + "-tg"
	templateName := cfg.NamePrefix + "-tmpl-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostTemplateLinkResourceConfig(cfg, hgName, hostName, tgName, templateName),
			},
			{
				ResourceName:      "zabbix_host_template_link.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccHostTemplateLinkResource_OnDestroyUnlink(t *testing.T) {
	cfg := testhelper.Setup(t)
	hgName := cfg.NamePrefix + "-hg"
	hostName := cfg.NamePrefix + "-host-unlink"
	tgName := cfg.NamePrefix + "-tg"
	templateName := cfg.NamePrefix + "-tmpl-unlink"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHostTemplateLinkResourceWithOnDestroyConfig(cfg, hgName, hostName, tgName, templateName, "unlink"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"zabbix_host_template_link.test",
						tfjsonpath.New("on_destroy"),
						knownvalue.StringExact("unlink"),
					),
				},
			},
			// Delete is exercised automatically by TestCase
		},
	})
}

func testAccHostTemplateLinkResourceConfig(cfg *testhelper.Config, hgName, hostName, tgName, templateName string) string {
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

resource "zabbix_template_group" "test" {
  name = %[5]q
}

resource "zabbix_template" "test" {
  host               = %[6]q
  template_group_ids = [zabbix_template_group.test.id]
}

resource "zabbix_host_template_link" "test" {
  host_id     = zabbix_host.test.id
  template_id = zabbix_template.test.id
}
`, cfg.URL, cfg.Token, hgName, hostName, tgName, templateName)
}

func testAccHostTemplateLinkResourceWithOnDestroyConfig(cfg *testhelper.Config, hgName, hostName, tgName, templateName, onDestroy string) string {
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

resource "zabbix_template_group" "test" {
  name = %[5]q
}

resource "zabbix_template" "test" {
  host               = %[6]q
  template_group_ids = [zabbix_template_group.test.id]
}

resource "zabbix_host_template_link" "test" {
  host_id     = zabbix_host.test.id
  template_id = zabbix_template.test.id
  on_destroy  = %[7]q
}
`, cfg.URL, cfg.Token, hgName, hostName, tgName, templateName, onDestroy)
}
