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

func TestAccMediaTypeDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaTypeDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_media_type.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("data.zabbix_media_type.test", tfjsonpath.New("type"), knownvalue.StringExact("email")),
				},
			},
		},
	})
}

func TestAccMediaTypeDataSource_ByName(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-ds-name"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaTypeDataSourceByNameConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_media_type.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_media_type.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccMediaTypeDataSource_ZeroMatchError(t *testing.T) {
	cfg := testhelper.Setup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccMediaTypeDataSourceByNameOnlyConfig(cfg, cfg.NamePrefix+"-nonexistent"),
				ExpectError: regexp.MustCompile(`Media type not found`),
			},
		},
	})
}

// ---- Unit tests ----

func TestMediaTypeDataSource_MultipleMatchError(t *testing.T) {
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"mediatypeid": "1", "name": "Email", "type": "0", "status": "0", "maxsessions": "1", "maxattempts": "3"},
			{"mediatypeid": "2", "name": "Email", "type": "0", "status": "0", "maxsessions": "1", "maxattempts": "3"},
		},
	}

	ds := newFakeMediaTypeDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildMediaTypeDataSourceConfig(t, "", "Email")}
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

func TestMediaTypeDataSource_MissingKeyError(t *testing.T) {
	ds := newFakeMediaTypeDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildMediaTypeDataSourceConfig(t, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor name is set")
	}
}

// ---- helpers ----

func newFakeMediaTypeDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewMediaTypeDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildMediaTypeDataSourceConfig(t *testing.T, id, name string) tfsdk.Config {
	t.Helper()

	toStr := func(s string) tftypes.Value {
		if s == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, s)
	}
	nullStr := tftypes.NewValue(tftypes.String, nil)
	nullNum := tftypes.NewValue(tftypes.Number, nil)

	webhookParamType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String, "value": tftypes.String}}
	webhookSettingsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"script": tftypes.String, "timeout": tftypes.String,
		"process_tags": tftypes.Bool, "show_event_menu": tftypes.Bool,
		"event_menu_url": tftypes.String, "event_menu_name": tftypes.String,
		"parameters": tftypes.List{ElementType: webhookParamType},
	}}
	emailSettingsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"smtp_server": tftypes.String, "smtp_port": tftypes.Number,
		"smtp_helo": tftypes.String, "smtp_email": tftypes.String,
		"smtp_security": tftypes.String, "smtp_authentication": tftypes.String,
		"username": tftypes.String, "password": tftypes.String, "content_type": tftypes.String,
	}}
	smsSettingsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"gsm_modem": tftypes.String}}
	scriptSettingsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"exec_path": tftypes.String, "exec_params": tftypes.String}}
	msgTemplateType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"eventsource": tftypes.Number, "recovery": tftypes.Number, "subject": tftypes.String, "message": tftypes.String,
	}}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String, "name": tftypes.String,
		"type": tftypes.String, "status": tftypes.String, "description": tftypes.String,
		"max_sessions": tftypes.Number, "max_attempts": tftypes.Number, "attempt_interval": tftypes.String,
		"email_settings": emailSettingsType, "sms_settings": smsSettingsType,
		"script_settings": scriptSettingsType, "webhook_settings": webhookSettingsType,
		"message_templates": tftypes.List{ElementType: msgTemplateType},
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewMediaTypeDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id": toStr(id), "name": toStr(name),
			"type": nullStr, "status": nullStr, "description": nullStr,
			"max_sessions": nullNum, "max_attempts": nullNum, "attempt_interval": nullStr,
			"email_settings":    tftypes.NewValue(emailSettingsType, nil),
			"sms_settings":      tftypes.NewValue(smsSettingsType, nil),
			"script_settings":   tftypes.NewValue(scriptSettingsType, nil),
			"webhook_settings":  tftypes.NewValue(webhookSettingsType, nil),
			"message_templates": tftypes.NewValue(tftypes.List{ElementType: msgTemplateType}, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccMediaTypeDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type" "seed" {
  name = %[3]q
  type = "email"

  email_settings = {
    smtp_server = "smtp.example.com"
    smtp_email  = "alerts@example.com"
  }
}

data "zabbix_media_type" "test" {
  id = zabbix_media_type.seed.id
}
`, cfg.URL, cfg.Token, name)
}

func testAccMediaTypeDataSourceByNameConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type" "seed" {
  name = %[3]q
  type = "email"

  email_settings = {
    smtp_server = "smtp.example.com"
    smtp_email  = "alerts@example.com"
  }
}

data "zabbix_media_type" "test" {
  name       = zabbix_media_type.seed.name
  depends_on = [zabbix_media_type.seed]
}
`, cfg.URL, cfg.Token, name)
}

func testAccMediaTypeDataSourceByNameOnlyConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

data "zabbix_media_type" "test" {
  name = %[3]q
}
`, cfg.URL, cfg.Token, name)
}
