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

func TestAccUserDirectorySAMLDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-saml-ds"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_user_directory_saml.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_saml.test", tfjsonpath.New("idp_entityid"), knownvalue.StringExact("http://idp.example.com/metadata")),
					statecheck.ExpectKnownValue("data.zabbix_user_directory_saml.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccSAMLDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_saml" "seed" {
  name         = %q
  idp_entityid = "http://idp.example.com/metadata"
  sp_entityid  = "zabbix"
}

data "zabbix_user_directory_saml" "test" {
  name       = zabbix_user_directory_saml.seed.name
  depends_on = [zabbix_user_directory_saml.seed]
}
`, cfg.URL, cfg.Token, name)
}
