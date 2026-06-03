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

func TestAccMediaTypeScriptDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-script-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaTypeScriptDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_media_type_script.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_media_type_script.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("data.zabbix_media_type_script.test", tfjsonpath.New("exec_path"), knownvalue.StringExact("/usr/lib/zabbix/alertscripts/notify.sh")),
				},
			},
		},
	})
}

func testAccMediaTypeScriptDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_script" "seed" {
  name      = %[3]q
  exec_path = "/usr/lib/zabbix/alertscripts/notify.sh"
}

data "zabbix_media_type_script" "test" {
  id = zabbix_media_type_script.seed.id
}
`, cfg.URL, cfg.Token, name)
}
