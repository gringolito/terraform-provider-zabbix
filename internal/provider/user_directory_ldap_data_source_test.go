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

func TestAccUserDirectoryLDAPDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ldap-ds"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLDAPDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_user_directory_ldap.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_ldap.test", tfjsonpath.New("host"), knownvalue.StringExact("ldap.example.com")),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_ldap.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccLDAPDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_ldap" "seed" {
  name             = %q
  host             = "ldap.example.com"
  base_dn          = "dc=example,dc=com"
  search_attribute = "uid"
}

data "zabbix_user_directory_ldap" "test" {
  name       = zabbix_user_directory_ldap.seed.name
  depends_on = [zabbix_user_directory_ldap.seed]
}
`, cfg.URL, cfg.Token, name)
}
