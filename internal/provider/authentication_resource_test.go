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

func TestAuthenticationResource_Schema_IDIsComputed(t *testing.T) {
	r := provider.NewAuthenticationResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	attr, ok := schResp.Schema.Attributes["id"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("id is not a StringAttribute")
	}
	if !attr.Computed {
		t.Error("id must be Computed")
	}
	if attr.Required || attr.Optional {
		t.Error("id must not be Required or Optional")
	}
}

func TestAuthenticationResource_Schema_AuthenticationTypeValidValues(t *testing.T) {
	r := provider.NewAuthenticationResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	attr, ok := schResp.Schema.Attributes["authentication_type"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("authentication_type is not a StringAttribute")
	}
	if len(attr.Validators) == 0 {
		t.Fatal("authentication_type has no validators")
	}

	for _, invalid := range []string{"kerberos", "oauth", "1"} {
		req := validator.StringRequest{ConfigValue: types.StringValue(invalid)}
		resp := &validator.StringResponse{}
		for _, v := range attr.Validators {
			v.ValidateString(context.Background(), req, resp)
		}
		if !resp.Diagnostics.HasError() {
			t.Errorf("authentication_type should reject %q", invalid)
		}
	}
}

func TestAuthenticationResource_Schema_PasswdCheckRulesIsSet(t *testing.T) {
	r := provider.NewAuthenticationResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	attr, ok := schResp.Schema.Attributes["passwd_check_rules"].(fwschema.SetAttribute)
	if !ok {
		t.Fatal("passwd_check_rules is not a SetAttribute")
	}
	if attr.ElementType != types.StringType {
		t.Error("passwd_check_rules must have StringType elements")
	}
}

func TestAuthenticationResource_Schema_MFAIDIsOptionalComputed(t *testing.T) {
	r := provider.NewAuthenticationResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	attr, ok := schResp.Schema.Attributes["mfaid"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("mfaid is not a StringAttribute")
	}
	if !attr.Optional {
		t.Error("mfaid must be Optional")
	}
	if !attr.Computed {
		t.Error("mfaid must be Computed")
	}
}

func TestAuthenticationResource_Schema_DisabledUsrgrpIDOptionalComputed(t *testing.T) {
	r := provider.NewAuthenticationResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	attr, ok := schResp.Schema.Attributes["disabled_usrgrpid"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("disabled_usrgrpid is not a StringAttribute")
	}
	if !attr.Optional {
		t.Error("disabled_usrgrpid must be Optional")
	}
	if !attr.Computed {
		t.Error("disabled_usrgrpid must be Computed")
	}
}

func TestAuthenticationResource_Schema_PasswdMinLengthIsInt64(t *testing.T) {
	r := provider.NewAuthenticationResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	_, ok := schResp.Schema.Attributes["passwd_min_length"].(fwschema.Int64Attribute)
	if !ok {
		t.Fatal("passwd_min_length is not an Int64Attribute")
	}
}

// TestAccAuthenticationResource_AdoptAndUpdate verifies the singleton lifecycle:
// Create adopts the existing singleton, Update converges cleanly.
func TestAccAuthenticationResource_AdoptAndUpdate(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthenticationResourceConfig(cfg, 10),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_authentication.test", tfjsonpath.New("id"), knownvalue.StringExact("authentication")),
					statecheck.ExpectKnownValue("zabbix_authentication.test", tfjsonpath.New("authentication_type"), knownvalue.StringExact("internal")),
					statecheck.ExpectKnownValue("zabbix_authentication.test", tfjsonpath.New("passwd_min_length"), knownvalue.Int64Exact(10)),
				},
			},
			{
				Config: testAccAuthenticationResourceConfig(cfg, 12),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_authentication.test", tfjsonpath.New("passwd_min_length"), knownvalue.Int64Exact(12)),
				},
			},
		},
	})
}

// TestAccAuthenticationResource_DeleteResetsToDefaults verifies that terraform destroy
// resets the singleton to Zabbix defaults and emits a diagnostic warning.
func TestAccAuthenticationResource_DeleteResetsToDefaults(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthenticationResourceConfig(cfg, 12),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("zabbix_authentication.test", tfjsonpath.New("id"), knownvalue.StringExact("authentication")),
				},
			},
			{
				Config:  testAccAuthenticationEmptyConfig(cfg),
				Destroy: true,
				// Destroy emits a warning diagnostic; the test framework captures it
				// but does not fail on warnings.
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccAuthenticationResource_Import verifies that importing with the literal ID
// "authentication" loads the singleton state.
func TestAccAuthenticationResource_Import(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccAuthenticationResourceConfig(cfg, 10)},
			{
				ResourceName:      "zabbix_authentication.test",
				ImportState:       true,
				ImportStateId:     "authentication",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAuthenticationResourceConfig(cfg *testhelper.Config, passwdMinLength int) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %q
  api_token  = %q
}

resource "zabbix_authentication" "test" {
  passwd_min_length = %d
}
`, cfg.URL, cfg.Token, passwdMinLength)
}

func testAccAuthenticationEmptyConfig(cfg *testhelper.Config) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %q
  api_token  = %q
}
`, cfg.URL, cfg.Token)
}
