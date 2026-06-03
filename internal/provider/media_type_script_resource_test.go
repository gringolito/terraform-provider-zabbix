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

func TestAccMediaTypeScriptResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-mt-script"
	updated := cfg.NamePrefix + "-mt-script-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccScriptMediaTypeResourceConfig(cfg, initial, "/usr/lib/zabbix/alertscripts/notify.sh"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_script.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_media_type_script.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("zabbix_media_type_script.test", tfjsonpath.New("exec_path"), knownvalue.StringExact("/usr/lib/zabbix/alertscripts/notify.sh")),
				},
			},
			{
				Config: testAccScriptMediaTypeResourceConfig(cfg, updated, "/usr/lib/zabbix/alertscripts/notify-v2.sh"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_script.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
					statecheck.ExpectKnownValue("zabbix_media_type_script.test", tfjsonpath.New("exec_path"), knownvalue.StringExact("/usr/lib/zabbix/alertscripts/notify-v2.sh")),
				},
			},
		},
	})
}

func testAccScriptMediaTypeResourceConfig(cfg *testhelper.Config, name, execPath string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_script" "test" {
  name      = %[3]q
  exec_path = %[4]q
}
`, cfg.URL, cfg.Token, name, execPath)
}
