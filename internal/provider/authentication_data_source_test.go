package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAuthenticationDataSource_Schema_AllAttributesComputed(t *testing.T) {
	ds := provider.NewAuthenticationDataSource()
	schResp := &fwdatasource.SchemaResponse{}
	ds.Schema(context.Background(), fwdatasource.SchemaRequest{}, schResp)

	for name, attr := range schResp.Schema.Attributes {
		switch a := attr.(type) {
		case fwschema.StringAttribute:
			if !a.Computed {
				t.Errorf("attribute %q must be Computed", name)
			}
			if a.Required || a.Optional {
				t.Errorf("attribute %q must not be Required or Optional", name)
			}
		case fwschema.Int64Attribute:
			if !a.Computed {
				t.Errorf("attribute %q must be Computed", name)
			}
			if a.Required || a.Optional {
				t.Errorf("attribute %q must not be Required or Optional", name)
			}
		case fwschema.SetAttribute:
			if !a.Computed {
				t.Errorf("attribute %q must be Computed", name)
			}
			if a.Required || a.Optional {
				t.Errorf("attribute %q must not be Required or Optional", name)
			}
		}
	}
}

func TestAuthenticationDataSource_Schema_PasswdCheckRulesIsSet(t *testing.T) {
	ds := provider.NewAuthenticationDataSource()
	schResp := &fwdatasource.SchemaResponse{}
	ds.Schema(context.Background(), fwdatasource.SchemaRequest{}, schResp)

	attr, ok := schResp.Schema.Attributes["passwd_check_rules"].(fwschema.SetAttribute)
	if !ok {
		t.Fatal("passwd_check_rules is not a SetAttribute")
	}
	if attr.ElementType != types.StringType {
		t.Error("passwd_check_rules must have StringType elements")
	}
}

func TestAuthenticationDataSource_Schema_PasswdMinLengthIsInt64(t *testing.T) {
	ds := provider.NewAuthenticationDataSource()
	schResp := &fwdatasource.SchemaResponse{}
	ds.Schema(context.Background(), fwdatasource.SchemaRequest{}, schResp)

	_, ok := schResp.Schema.Attributes["passwd_min_length"].(fwschema.Int64Attribute)
	if !ok {
		t.Fatal("passwd_min_length is not an Int64Attribute")
	}
}

func TestAccAuthenticationDataSource_ReadsCurrentState(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAuthenticationDataSourceConfig(cfg, 11),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_authentication.current", tfjsonpath.New("id"), knownvalue.StringExact("authentication")),
					statecheck.ExpectKnownValue("data.zabbix_authentication.current", tfjsonpath.New("authentication_type"), knownvalue.StringExact("internal")),
					statecheck.ExpectKnownValue("data.zabbix_authentication.current", tfjsonpath.New("passwd_min_length"), knownvalue.Int64Exact(11)),
				},
			},
		},
	})
}

func testAccAuthenticationDataSourceConfig(cfg *testhelper.Config, passwdMinLength int) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %q
  api_token  = %q
}

resource "zabbix_authentication" "test" {
  passwd_min_length = %d
}

data "zabbix_authentication" "current" {
  depends_on = [zabbix_authentication.test]
}
`, cfg.URL, cfg.Token, passwdMinLength)
}
