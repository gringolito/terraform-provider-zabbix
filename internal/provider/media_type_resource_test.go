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

// ---- Email CRUD ----

func TestAccMediaTypeResource_EmailCRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-mt-email"
	updated := cfg.NamePrefix + "-mt-email-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccEmailMediaTypeConfig(cfg, initial, "smtp.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("type"), knownvalue.StringExact("email")),
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("status"), knownvalue.StringExact("enabled")),
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(
						"zabbix_media_type.test",
						tfjsonpath.New("email_settings").AtMapKey("smtp_server"),
						knownvalue.StringExact("smtp.example.com"),
					),
					statecheck.ExpectKnownValue(
						"zabbix_media_type.test",
						tfjsonpath.New("email_settings").AtMapKey("smtp_email"),
						knownvalue.StringExact("alerts@example.com"),
					),
				},
			},
			// Update name and SMTP server
			{
				Config: testAccEmailMediaTypeConfig(cfg, updated, "smtp2.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
					statecheck.ExpectKnownValue(
						"zabbix_media_type.test",
						tfjsonpath.New("email_settings").AtMapKey("smtp_server"),
						knownvalue.StringExact("smtp2.example.com"),
					),
				},
			},
			// Delete is automatic
		},
	})
}

func TestAccMediaTypeResource_EmailImport(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-email-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailMediaTypeConfig(cfg, name, "smtp.example.com"),
			},
			{
				ResourceName:      "zabbix_media_type.test",
				ImportState:       true,
				ImportStateVerify: true,
				// password is not returned by API, so skip it in verify
				ImportStateVerifyIgnore: []string{"email_settings.password"},
			},
		},
	})
}

func TestAccMediaTypeResource_EmailDrift(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-email-drift"
	renamed := cfg.NamePrefix + "-mt-email-drift-oob"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailMediaTypeConfig(cfg, name, "smtp.example.com"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_media_type.test"]
						if rs == nil {
							return fmt.Errorf("resource not found in state")
						}
						capturedID = rs.Primary.ID
						return nil
					},
				),
			},
			// Rename out-of-band, then verify apply reconciles
			{
				PreConfig: func() {
					c, err := client.New(context.Background(), cfg.URL, cfg.Token)
					if err != nil {
						t.Fatalf("drift PreConfig: client.New: %v", err)
					}
					mt, err := client.MediaTypeGet(context.Background(), c, capturedID)
					if err != nil || mt == nil {
						t.Fatalf("drift PreConfig: get media type: %v", err)
					}
					mt.Name = renamed
					if err := client.MediaTypeUpdate(context.Background(), c, *mt); err != nil {
						t.Fatalf("drift PreConfig: rename out-of-band: %v", err)
					}
				},
				Config: testAccEmailMediaTypeConfig(cfg, name, "smtp.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
				},
			},
		},
	})
}

// ---- Webhook CRUD ----

func TestAccMediaTypeResource_WebhookCRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-mt-wh"
	updated := cfg.NamePrefix + "-mt-wh-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccWebhookMediaTypeConfig(cfg, initial, "return 'OK';"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("type"), knownvalue.StringExact("webhook")),
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(
						"zabbix_media_type.test",
						tfjsonpath.New("webhook_settings").AtMapKey("script"),
						knownvalue.StringExact("return 'OK';"),
					),
				},
			},
			// Update name
			{
				Config: testAccWebhookMediaTypeConfig(cfg, updated, "return 'OK';"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
				},
			},
		},
	})
}

func TestAccMediaTypeResource_WebhookImport(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-wh-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookMediaTypeConfig(cfg, name, "return 'OK';"),
			},
			{
				ResourceName:      "zabbix_media_type.test",
				ImportState:       true,
				ImportStateVerify: true,
				// webhook parameter values are sensitive and not returned by API
				ImportStateVerifyIgnore: []string{"webhook_settings.parameters"},
			},
		},
	})
}

// ---- Config helpers ----

func testAccEmailMediaTypeConfig(cfg *testhelper.Config, name, smtpServer string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type" "test" {
  name = %[3]q
  type = "email"

  email_settings = {
    smtp_server = %[4]q
    smtp_email  = "alerts@example.com"
  }
}
`, cfg.URL, cfg.Token, name, smtpServer)
}

func testAccWebhookMediaTypeConfig(cfg *testhelper.Config, name, script string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type" "test" {
  name = %[3]q
  type = "webhook"

  webhook_settings = {
    script = %[4]q
  }
}
`, cfg.URL, cfg.Token, name, script)
}
