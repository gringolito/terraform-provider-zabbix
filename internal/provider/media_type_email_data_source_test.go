package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/clienttest"
	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	"github.com/gringolito/terraform-provider-zabbix/internal/testhelper"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// ---- Acceptance tests ----

func TestAccMediaTypeEmailDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-email-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaTypeEmailDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_media_type_email.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_media_type_email.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("data.zabbix_media_type_email.test", tfjsonpath.New("smtp_server"), knownvalue.StringExact("smtp.example.com")),
				},
			},
		},
	})
}

func TestAccMediaTypeEmailDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-email-ds-name"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaTypeEmailDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_media_type_email.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_media_type_email.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccMediaTypeEmailDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccMediaTypeEmailDataSourceByNameOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`Media type not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestMediaTypeEmailDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"mediatypeid": "1", "name": "Email", "type": "0", "status": "0", "maxsessions": "1", "maxattempts": "3", "attempt_interval": "10s", "description": "", "smtp_server": "", "smtp_port": "25", "smtp_helo": "", "smtp_email": "", "smtp_security": "0", "smtp_authentication": "0", "username": "", "passwd": "", "content_type": "1", "message_templates": []map[string]any{}},
			{"mediatypeid": "2", "name": "Email", "type": "0", "status": "0", "maxsessions": "1", "maxattempts": "3", "attempt_interval": "10s", "description": "", "smtp_server": "", "smtp_port": "25", "smtp_helo": "", "smtp_email": "", "smtp_security": "0", "smtp_authentication": "0", "username": "", "passwd": "", "content_type": "1", "message_templates": []map[string]any{}},
		},
	}

	ds := newFakeMediaTypeEmailDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildMediaTypeEmailDataSourceConfig(t, "", "Email")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for multiple matches, got none")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Multiple media types found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Multiple media types found' diagnostic, got: %s", resp.Diagnostics)
	}
}

func TestMediaTypeEmailDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeMediaTypeEmailDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildMediaTypeEmailDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor name is set")
	}
}

// ---- Helpers ----

func newFakeMediaTypeEmailDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewMediaTypeEmailDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildMediaTypeEmailDataSourceConfig(t *testing.T, id, name string) tfsdk.Config {
	t.Helper()

	toStr := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}
	nullStr := tftypes.NewValue(tftypes.String, nil)
	nullNum := tftypes.NewValue(tftypes.Number, nil)
	msgTemplateType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"event_source": tftypes.String, "recovery": tftypes.String, "subject": tftypes.String, "message": tftypes.String,
	}}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String, "name": tftypes.String,
		"status": tftypes.String, "description": tftypes.String, "attempt_interval": tftypes.String,
		"max_sessions": tftypes.Number, "max_attempts": tftypes.Number,
		"smtp_server": tftypes.String, "smtp_port": tftypes.Number,
		"smtp_helo": tftypes.String, "smtp_email": tftypes.String,
		"smtp_security": tftypes.String, "smtp_authentication": tftypes.String,
		"username": tftypes.String, "content_type": tftypes.String,
		"message_templates": tftypes.List{ElementType: msgTemplateType},
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewMediaTypeEmailDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id": toStr(id), "name": toStr(name),
			"status": nullStr, "description": nullStr, "attempt_interval": nullStr,
			"max_sessions": nullNum, "max_attempts": nullNum,
			"smtp_server": nullStr, "smtp_port": nullNum,
			"smtp_helo": nullStr, "smtp_email": nullStr,
			"smtp_security": nullStr, "smtp_authentication": nullStr,
			"username": nullStr, "content_type": nullStr,
			"message_templates": tftypes.NewValue(tftypes.List{ElementType: msgTemplateType}, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccMediaTypeEmailDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_email" "seed" {
  name        = %[3]q
  smtp_server = "smtp.example.com"
  smtp_email  = "alerts@example.com"
}

data "zabbix_media_type_email" "test" {
  id = zabbix_media_type_email.seed.id
}
`, cfg.URL, cfg.Token, name)
}

func testAccMediaTypeEmailDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_email" "seed" {
  name        = %[3]q
  smtp_server = "smtp.example.com"
  smtp_email  = "alerts@example.com"
}

data "zabbix_media_type_email" "test" {
  name       = zabbix_media_type_email.seed.name
  depends_on = [zabbix_media_type_email.seed]
}
`, cfg.URL, cfg.Token, name)
}

func testAccMediaTypeEmailDataSourceByNameOnlyConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_media_type_email" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
