package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestUserDirectoryLDAPResource_PortBelowRange(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccLDAPResourcePortBelowRangeConfig(cfg),
				ExpectError: regexp.MustCompile(`between 1 and 65535`),
			},
		},
	})
}

func TestUserDirectoryLDAPResource_PortAboveRange(t *testing.T) {
	cfg := &testhelper.Config{URL: "http://fake:8080", Token: "fake"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccLDAPResourcePortAboveRangeConfig(cfg),
				ExpectError: regexp.MustCompile(`between 1 and 65535`),
			},
		},
	})
}

func TestUserDirectoryLDAPResource_SchemaValidation(t *testing.T) {
	r := provider.NewUserDirectoryLDAPResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	t.Run("bind_password is sensitive", func(t *testing.T) {
		attr, ok := schResp.Schema.Attributes["bind_password"].(fwschema.StringAttribute)
		if !ok {
			t.Fatal("bind_password is not a StringAttribute")
		}
		if !attr.Sensitive {
			t.Error("bind_password must be Sensitive")
		}
	})

	t.Run("required fields present", func(t *testing.T) {
		for _, name := range []string{"host", "base_dn", "search_attribute"} {
			attr, ok := schResp.Schema.Attributes[name].(fwschema.StringAttribute)
			if !ok {
				t.Errorf("%s is not a StringAttribute", name)
				continue
			}
			if !attr.Required {
				t.Errorf("%s must be Required", name)
			}
		}
	})

	t.Run("start_tls rejects unknown values", func(t *testing.T) {
		attr, ok := schResp.Schema.Attributes["start_tls"].(fwschema.StringAttribute)
		if !ok {
			t.Fatal("start_tls is not a StringAttribute")
		}
		if len(attr.Validators) == 0 {
			t.Fatal("start_tls has no validators")
		}
		req := validator.StringRequest{ConfigValue: types.StringValue("yes")}
		resp := &validator.StringResponse{}
		for _, v := range attr.Validators {
			v.ValidateString(context.Background(), req, resp)
		}
		if !resp.Diagnostics.HasError() {
			t.Error("start_tls should reject 'yes'")
		}
	})

	t.Run("provision_status rejects unknown values", func(t *testing.T) {
		attr, ok := schResp.Schema.Attributes["provision_status"].(fwschema.StringAttribute)
		if !ok {
			t.Fatal("provision_status is not a StringAttribute")
		}
		req := validator.StringRequest{ConfigValue: types.StringValue("maybe")}
		resp := &validator.StringResponse{}
		for _, v := range attr.Validators {
			v.ValidateString(context.Background(), req, resp)
		}
		if !resp.Diagnostics.HasError() {
			t.Error("provision_status should reject 'maybe'")
		}
	})
}

func TestAccUserDirectoryLDAPResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-ldap"
	updated := cfg.NamePrefix + "-ldap-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLDAPResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("host"), knownvalue.StringExact("ldap.example.com")),
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("port"), knownvalue.Int64Exact(389)),
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccLDAPResourceConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_ldap.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
				},
			},
		},
	})
}

func TestAccUserDirectoryLDAPResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-ldap-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccLDAPResourceConfig(cfg, name)},
			{
				ResourceName:            "zabbix_user_directory_ldap.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"bind_password"},
			},
		},
	})
}

func testAccLDAPResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %q
  api_token  = %q
}

resource "zabbix_user_directory_ldap" "test" {
  name             = %q
  host             = "ldap.example.com"
  base_dn          = "dc=example,dc=com"
  search_attribute = "uid"
}
`, cfg.URL, cfg.Token, name)
}

func testAccLDAPResourcePortBelowRangeConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %q
  api_token  = %q
}

resource "zabbix_user_directory_ldap" "test" {
  name             = "test"
  host             = "ldap.example.com"
  base_dn          = "dc=example,dc=com"
  search_attribute = "uid"
  port             = 0
}
`, cfg.URL, cfg.Token)
}

func testAccLDAPResourcePortAboveRangeConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %q
  api_token  = %q
}

resource "zabbix_user_directory_ldap" "test" {
  name             = "test"
  host             = "ldap.example.com"
  base_dn          = "dc=example,dc=com"
  search_attribute = "uid"
  port             = 65536
}
`, cfg.URL, cfg.Token)
}
