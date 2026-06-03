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

func TestAccMediaTypeWebhookResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-mt-wh"
	updated := cfg.NamePrefix + "-mt-wh-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookMediaTypeResourceConfig(cfg, initial, "return 'OK';"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_webhook.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_media_type_webhook.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("zabbix_media_type_webhook.test", tfjsonpath.New("script"), knownvalue.StringExact("return 'OK';")),
				},
			},
			{
				Config: testAccWebhookMediaTypeResourceConfig(cfg, updated, "return 'OK';"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_webhook.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
				},
			},
		},
	})
}

func TestAccMediaTypeWebhookResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-wh-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookMediaTypeResourceConfig(cfg, name, "return 'OK';"),
			},
			{
				ResourceName:            "zabbix_media_type_webhook.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parameters"},
			},
		},
	})
}

func testAccWebhookMediaTypeResourceConfig(cfg *testhelper.Config, name, script string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_webhook" "test" {
  name   = %[3]q
  script = %[4]q
}
`, cfg.URL, cfg.Token, name, script)
}
