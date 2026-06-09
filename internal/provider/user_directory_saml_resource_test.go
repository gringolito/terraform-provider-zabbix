package provider_test

import (
	"context"
	"fmt"
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

func TestUserDirectorySAMLResource_SchemaValidation(t *testing.T) {
	r := provider.NewUserDirectorySAMLResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	t.Run("required fields present", func(t *testing.T) {
		for _, name := range []string{"idp_entityid", "sp_entityid"} {
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

	t.Run("boolean flag fields reject unknown values", func(t *testing.T) {
		for _, name := range []string{"sign_messages", "sign_assertions", "encrypt_nameid", "scim_status"} {
			attr, ok := schResp.Schema.Attributes[name].(fwschema.StringAttribute)
			if !ok {
				t.Errorf("%s is not a StringAttribute", name)
				continue
			}
			if len(attr.Validators) == 0 {
				t.Errorf("%s has no validators", name)
				continue
			}
			req := validator.StringRequest{ConfigValue: types.StringValue("yes")}
			resp := &validator.StringResponse{}
			for _, v := range attr.Validators {
				v.ValidateString(context.Background(), req, resp)
			}
			if !resp.Diagnostics.HasError() {
				t.Errorf("%s should reject 'yes'", name)
			}
		}
	})
}

func TestAccUserDirectorySAMLResource_CRUD(t *testing.T) {
	cfg := testhelper.Setup(t)
	initial := cfg.NamePrefix + "-saml"
	updated := cfg.NamePrefix + "-saml-upd"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLResourceConfig(cfg, initial),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("name"), knownvalue.StringExact(initial)),
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("idp_entityid"), knownvalue.StringExact("http://idp.example.com/metadata")),
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccSAMLResourceConfig(cfg, updated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_user_directory_saml.test", tfjsonpath.New("name"), knownvalue.StringExact(updated)),
				},
			},
		},
	})
}

func TestAccUserDirectorySAMLResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-saml-imp"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccSAMLResourceConfig(cfg, name)},
			{
				ResourceName:      "zabbix_user_directory_saml.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSAMLResourceConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  url   = %q
  token = %q
}

resource "zabbix_user_directory_saml" "test" {
  name         = %q
  idp_entityid = "http://idp.example.com/metadata"
  sp_entityid  = "zabbix"
}
`, cfg.URL, cfg.Token, name)
}
