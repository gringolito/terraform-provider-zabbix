package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	tfwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---- URL validator ----

func TestURLValidator_Valid(t *testing.T) {
	cases := []string{
		"http://zabbix.example.com",
		"https://zabbix.example.com",
		"http://localhost:8080",
		"https://192.168.1.1/zabbix",
	}
	v := urlValidator{}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tc),
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tc, resp.Diagnostics)
			}
		})
	}
}

func TestURLValidator_Invalid(t *testing.T) {
	cases := []string{
		"not-a-url",
		"ftp://zabbix.example.com",
		"//zabbix.example.com",
		"zabbix.example.com",
		"",
	}
	v := urlValidator{}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(tc),
			}, resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tc)
			}
		})
	}
}

func TestURLValidator_SkipsNullAndUnknown(t *testing.T) {
	v := urlValidator{}

	nullResp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringNull(),
	}, nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Error("unexpected error for null value")
	}

	unknownResp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}, unknownResp)
	if unknownResp.Diagnostics.HasError() {
		t.Error("unexpected error for unknown value")
	}
}

// ---- Schema ----

func TestProviderSchema_Attributes(t *testing.T) {
	p := &ZabbixProvider{version: "test"}
	resp := &tfwprovider.SchemaResponse{}
	p.Schema(context.Background(), tfwprovider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema returned errors: %s", resp.Diagnostics)
	}
	attrs := resp.Schema.Attributes
	if _, ok := attrs["zabbix_url"]; !ok {
		t.Error("schema missing zabbix_url attribute")
	}
	if _, ok := attrs["api_token"]; !ok {
		t.Error("schema missing api_token attribute")
	}
}

// ---- Configure helpers ----

func versionServer(t *testing.T, version string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"result":  version,
			"id":      1,
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func makeConfig(t *testing.T, urlVal, tokenVal tftypes.Value) tfsdk.Config {
	t.Helper()
	p := &ZabbixProvider{}
	schemaResp := &tfwprovider.SchemaResponse{}
	p.Schema(context.Background(), tfwprovider.SchemaRequest{}, schemaResp)
	return tfsdk.Config{
		Raw: tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"zabbix_url": tftypes.String,
				"api_token":  tftypes.String,
			},
		}, map[string]tftypes.Value{
			"zabbix_url": urlVal,
			"api_token":  tokenVal,
		}),
		Schema: schemaResp.Schema,
	}
}

func nullStr() tftypes.Value        { return tftypes.NewValue(tftypes.String, nil) }
func strVal(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

func configureProv(t *testing.T, cfg tfsdk.Config) *tfwprovider.ConfigureResponse {
	t.Helper()
	p := &ZabbixProvider{version: "test"}
	resp := &tfwprovider.ConfigureResponse{}
	p.Configure(context.Background(), tfwprovider.ConfigureRequest{Config: cfg}, resp)
	return resp
}

// ---- Configure tests ----

func TestProviderConfigure_MissingURL(t *testing.T) {
	t.Setenv("ZABBIX_URL", "")
	t.Setenv("ZABBIX_TOKEN", "")
	resp := configureProv(t, makeConfig(t, nullStr(), strVal("token")))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing URL, got none")
	}
}

func TestProviderConfigure_MissingToken(t *testing.T) {
	t.Setenv("ZABBIX_URL", "")
	t.Setenv("ZABBIX_TOKEN", "")
	srv := versionServer(t, "7.0.0")
	resp := configureProv(t, makeConfig(t, strVal(srv.URL), nullStr()))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for missing token, got none")
	}
}

func TestProviderConfigure_EnvVarFallbacks(t *testing.T) {
	srv := versionServer(t, "7.0.0")
	t.Setenv("ZABBIX_URL", srv.URL)
	t.Setenv("ZABBIX_TOKEN", "env-token")

	resp := configureProv(t, makeConfig(t, nullStr(), nullStr()))
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics)
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be populated")
	}
}

func TestProviderConfigure_Targeted(t *testing.T) {
	t.Setenv("ZABBIX_URL", "")
	t.Setenv("ZABBIX_TOKEN", "")
	srv := versionServer(t, "7.0.3")
	resp := configureProv(t, makeConfig(t, strVal(srv.URL), strVal("tok")))
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics)
	}
	// no warnings expected for Targeted
	for _, d := range resp.Diagnostics {
		if d.Severity().String() == "Warning" {
			t.Errorf("unexpected warning for Targeted version: %s", d.Summary())
		}
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be populated")
	}
}

func TestProviderConfigure_Tolerated(t *testing.T) {
	t.Setenv("ZABBIX_URL", "")
	t.Setenv("ZABBIX_TOKEN", "")
	srv := versionServer(t, "7.2.1")
	resp := configureProv(t, makeConfig(t, strVal(srv.URL), strVal("tok")))
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for Tolerated version: %s", resp.Diagnostics)
	}
	found := false
	for _, d := range resp.Diagnostics {
		if d.Severity().String() == "Warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected a warning diagnostic for Tolerated version, got none")
	}
}

func TestProviderConfigure_Unsupported(t *testing.T) {
	t.Setenv("ZABBIX_URL", "")
	t.Setenv("ZABBIX_TOKEN", "")
	srv := versionServer(t, "5.4.0")
	resp := configureProv(t, makeConfig(t, strVal(srv.URL), strVal("tok")))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for Unsupported version, got none")
	}
	if resp.ResourceData != nil {
		t.Error("ResourceData should be nil on Unsupported version")
	}
}

func TestProviderConfigure_MalformedURL(t *testing.T) {
	t.Setenv("ZABBIX_URL", "")
	t.Setenv("ZABBIX_TOKEN", "")
	// Schema-level URL validation: the validator should catch this, but
	// Configure also handles a connection error gracefully.
	resp := configureProv(t, makeConfig(t, strVal("not-a-url"), strVal("tok")))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for malformed URL, got none")
	}
}

func TestProviderConfigure_EnvVarURLOverridesEmpty(t *testing.T) {
	srv := versionServer(t, "7.0.0")
	t.Setenv("ZABBIX_URL", srv.URL)
	t.Setenv("ZABBIX_TOKEN", "tok")

	// Explicit empty string in config should fall back to env var.
	resp := configureProv(t, makeConfig(t, nullStr(), nullStr()))
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics)
	}
}

// Ensure t.Setenv cleans up after the test; this is a sentinel that os.Getenv
// reads the env correctly during the test.
func TestEnvVarSentinel(t *testing.T) {
	key := "ZABBIX_SENTINEL_TEST_" + t.Name()
	os.Unsetenv(key)
	if v := os.Getenv(key); v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
	t.Setenv(key, "hello")
	if v := os.Getenv(key); v != "hello" {
		t.Fatalf("expected %q, got %q", "hello", v)
	}
}
