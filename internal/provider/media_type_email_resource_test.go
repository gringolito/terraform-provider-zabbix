package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMediaTypeEmailResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-mt-email"
	updated := cfg.NamePrefix + "-mt-email-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailMediaTypeResourceConfig(cfg, initial, "smtp.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("status"), knownvalue.StringExact("enabled")),
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("smtp_server"), knownvalue.StringExact("smtp.example.com")),
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("smtp_email"), knownvalue.StringExact("alerts@example.com")),
				},
			},
			{
				Config: testAccEmailMediaTypeResourceConfig(cfg, updated, "smtp2.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("smtp_server"), knownvalue.StringExact("smtp2.example.com")),
				},
			},
		},
	})
}

func TestAccMediaTypeEmailResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-email-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailMediaTypeResourceConfig(cfg, name, "smtp.example.com"),
			},
			{
				ResourceName:            "zabbix_media_type_email.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccMediaTypeEmailResource_Drift(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-email-drift"
	renamed := cfg.NamePrefix + "-mt-email-drift-oob"

	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailMediaTypeResourceConfig(cfg, name, "smtp.example.com"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["zabbix_media_type_email.test"]
						if rs == nil {
							return fmt.Errorf("resource not found in state")
						}
						capturedID = rs.Primary.ID
						return nil
					},
				),
			},
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
				Config: testAccEmailMediaTypeResourceConfig(cfg, name, "smtp.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_media_type_email.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
				},
			},
		},
	})
}

func TestMediaTypeEmailResource_SmtpPortBelowRange(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccEmailMediaTypeResourceSmtpPortBelowRangeConfig(cfg),
				ExpectError: regexp.MustCompile(`between 1 and 65535`),
			},
		},
	})
}

func TestMediaTypeEmailResource_SmtpPortAboveRange(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccEmailMediaTypeResourceSmtpPortAboveRangeConfig(cfg),
				ExpectError: regexp.MustCompile(`between 1 and 65535`),
			},
		},
	})
}

func testAccEmailMediaTypeResourceConfig(cfg *testhelper.Config, name, smtpServer string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_email" "test" {
  name        = %[3]q
  smtp_server = %[4]q
  smtp_email  = "alerts@example.com"
}
`, cfg.URL, cfg.Token, name, smtpServer)
}

func testAccEmailMediaTypeResourceSmtpPortBelowRangeConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_email" "test" {
  name        = "test"
  smtp_server = "smtp.example.com"
  smtp_email  = "alerts@example.com"
  smtp_port   = 0
}
`, cfg.URL, cfg.Token)
}

func testAccEmailMediaTypeResourceSmtpPortAboveRangeConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_email" "test" {
  name        = "test"
  smtp_server = "smtp.example.com"
  smtp_email  = "alerts@example.com"
  smtp_port   = 65536
}
`, cfg.URL, cfg.Token)
}
