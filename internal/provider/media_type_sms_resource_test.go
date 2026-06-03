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

func TestAccMediaTypeSMSResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-mt-sms"
	updated := cfg.NamePrefix + "-mt-sms-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSMSMediaTypeResourceConfig(cfg, initial, "/dev/ttyS0"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_sms.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_media_type_sms.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("zabbix_media_type_sms.test", tfjsonpath.New("gsm_modem"), knownvalue.StringExact("/dev/ttyS0")),
				},
			},
			{
				Config: testAccSMSMediaTypeResourceConfig(cfg, updated, "/dev/ttyS1"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_sms.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
					statecheck.ExpectKnownValue("zabbix_media_type_sms.test", tfjsonpath.New("gsm_modem"), knownvalue.StringExact("/dev/ttyS1")),
				},
			},
		},
	})
}

func testAccSMSMediaTypeResourceConfig(cfg *testhelper.Config, name, gsmModem string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_sms" "test" {
  name      = %[3]q
  gsm_modem = %[4]q
}
`, cfg.URL, cfg.Token, name, gsmModem)
}
