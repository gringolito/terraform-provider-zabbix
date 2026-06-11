package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

func TestConfigurationImport_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"configuration.import": rpcOK(t, true),
	})
	rules := client.ImportRules{
		Templates:      client.ImportRuleCreateUpdate{CreateMissing: true, UpdateExisting: true},
		TemplateGroups: client.ImportRuleCreateUpdate{CreateMissing: true, UpdateExisting: true},
	}
	if err := client.ConfigurationImport(t.Context(), c, "xml", "<zabbix_export/>", rules); err != nil {
		t.Fatalf("ConfigurationImport: %v", err)
	}
}

func TestConfigurationImport_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"configuration.import": rpcErr(t, -32602, "Invalid params."),
	})
	rules := client.ImportRules{}
	err := client.ConfigurationImport(t.Context(), c, "xml", "<invalid/>", rules)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConfigurationImport_ServerReturnsFalse(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"configuration.import": rpcOK(t, false),
	})
	rules := client.ImportRules{}
	err := client.ConfigurationImport(t.Context(), c, "xml", "<zabbix_export/>", rules)
	if err == nil {
		t.Fatal("expected error when server returns false, got nil")
	}
}
