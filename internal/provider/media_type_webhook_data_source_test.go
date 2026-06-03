package provider_test

import (
	"context"
	"fmt"
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

func TestAccMediaTypeWebhookDataSource_ByID(t *testing.T) {
	cfg := testhelper.Setup(t)
	name := cfg.NamePrefix + "-mt-wh-ds-id"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMediaTypeWebhookDataSourceByIDConfig(cfg, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.zabbix_media_type_webhook.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("data.zabbix_media_type_webhook.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("data.zabbix_media_type_webhook.test", tfjsonpath.New("script"), knownvalue.StringExact("return 'OK';")),
				},
			},
		},
	})
}

// ---- Unit tests ----

func TestMediaTypeWebhookDataSource_MultipleMatchError(t *testing.T) {
	webhookParam := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String}}
	fake := &clienttest.TestClient{
		Response: []map[string]any{
			{"mediatypeid": "1", "name": "Webhook", "type": "4", "status": "0", "maxsessions": "1", "maxattempts": "3", "attempt_interval": "10s", "description": "", "script": "", "timeout": "30s", "process_tags": "0", "show_event_menu": "0", "event_menu_url": "", "event_menu_name": "", "parameters": []map[string]any{}, "message_templates": []map[string]any{}},
			{"mediatypeid": "2", "name": "Webhook", "type": "4", "status": "0", "maxsessions": "1", "maxattempts": "3", "attempt_interval": "10s", "description": "", "script": "", "timeout": "30s", "process_tags": "0", "show_event_menu": "0", "event_menu_url": "", "event_menu_name": "", "parameters": []map[string]any{}, "message_templates": []map[string]any{}},
		},
	}

	ds := newFakeMediaTypeWebhookDataSource(t, fake)
	req := datasource.ReadRequest{Config: buildMediaTypeWebhookDataSourceConfig(t, webhookParam, "", "Webhook")}
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

func TestMediaTypeWebhookDataSource_MissingKeyError(t *testing.T) {
	webhookParam := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String}}
	ds := newFakeMediaTypeWebhookDataSource(t, &clienttest.TestClient{})
	req := datasource.ReadRequest{Config: buildMediaTypeWebhookDataSourceConfig(t, webhookParam, "", "")}
	resp := &datasource.ReadResponse{}
	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when neither id nor name is set")
	}
}

// ---- Helpers ----

func newFakeMediaTypeWebhookDataSource(t *testing.T, fake any) datasource.DataSource {
	t.Helper()
	ds := provider.NewMediaTypeWebhookDataSource()
	if c, ok := ds.(datasource.DataSourceWithConfigure); ok {
		cfgResp := &datasource.ConfigureResponse{}
		c.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: fake}, cfgResp)
		if cfgResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %s", cfgResp.Diagnostics)
		}
	}
	return ds
}

func buildMediaTypeWebhookDataSourceConfig(t *testing.T, webhookParam tftypes.Object, id, name string) tfsdk.Config {
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
		"eventsource": tftypes.String, "recovery": tftypes.String, "subject": tftypes.String, "message": tftypes.String,
	}}

	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String, "name": tftypes.String,
		"status": tftypes.String, "description": tftypes.String, "attempt_interval": tftypes.String,
		"max_sessions": tftypes.Number, "max_attempts": tftypes.Number,
		"script": tftypes.String, "timeout": tftypes.String,
		"process_tags": tftypes.Bool, "show_event_menu": tftypes.Bool,
		"event_menu_url": tftypes.String, "event_menu_name": tftypes.String,
		"parameters":        tftypes.List{ElementType: webhookParam},
		"message_templates": tftypes.List{ElementType: msgTemplateType},
	}}

	schemaResp := &datasource.SchemaResponse{}
	provider.NewMediaTypeWebhookDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	return tfsdk.Config{
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id": toStr(id), "name": toStr(name),
			"status": nullStr, "description": nullStr, "attempt_interval": nullStr,
			"max_sessions": nullNum, "max_attempts": nullNum,
			"script": nullStr, "timeout": nullStr,
			"process_tags":      tftypes.NewValue(tftypes.Bool, nil),
			"show_event_menu":   tftypes.NewValue(tftypes.Bool, nil),
			"event_menu_url":    nullStr,
			"event_menu_name":   nullStr,
			"parameters":        tftypes.NewValue(tftypes.List{ElementType: webhookParam}, nil),
			"message_templates": tftypes.NewValue(tftypes.List{ElementType: msgTemplateType}, nil),
		}),
		Schema: schemaResp.Schema,
	}
}

func testAccMediaTypeWebhookDataSourceByIDConfig(cfg *testhelper.Config, name string) string {
	return fmt.Sprintf(`
provider "zabbix" {
  zabbix_url = %[1]q
  api_token  = %[2]q
}

resource "zabbix_media_type_webhook" "seed" {
  name   = %[3]q
  script = "return 'OK';"
}

data "zabbix_media_type_webhook" "test" {
  id = zabbix_media_type_webhook.seed.id
}
`, cfg.URL, cfg.Token, name)
}
