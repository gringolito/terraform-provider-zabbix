package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- HostInterfaceCreate ----

func TestHostInterfaceCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.create": rpcOK(t, map[string]any{"interfaceids": []string{"42"}}),
	})
	iface := client.HostInterface{
		HostID: "10084",
		Type:   1,
		UseIP:  1,
		IP:     "192.168.1.1",
		DNS:    "",
		Port:   "10050",
		Main:   1,
	}
	id, err := client.HostInterfaceCreate(t.Context(), c, iface)
	if err != nil {
		t.Fatalf("HostInterfaceCreate: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want %q", id, "42")
	}
}

func TestHostInterfaceCreate_SNMP_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.create": rpcOK(t, map[string]any{"interfaceids": []string{"55"}}),
	})
	iface := client.HostInterface{
		HostID: "10084",
		Type:   2,
		UseIP:  1,
		IP:     "192.168.1.1",
		DNS:    "",
		Port:   "161",
		Main:   1,
		Details: &client.HostInterfaceSNMPDetails{
			Version:      2,
			Community:    "public",
			BulkRequests: 1,
		},
	}
	id, err := client.HostInterfaceCreate(t.Context(), c, iface)
	if err != nil {
		t.Fatalf("HostInterfaceCreate SNMP: %v", err)
	}
	if id != "55" {
		t.Errorf("id = %q, want %q", id, "55")
	}
}

func TestHostInterfaceCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.create": rpcErr(t, -32602, "Invalid params."),
	})
	iface := client.HostInterface{HostID: "1", Type: 1, Port: "10050"}
	_, err := client.HostInterfaceCreate(t.Context(), c, iface)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostInterfaceGet ----

func TestHostInterfaceGet_AgentType(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{{
			"interfaceid": "42",
			"hostid":      "10084",
			"type":        "1",
			"useip":       "1",
			"ip":          "192.168.1.1",
			"dns":         "",
			"port":        "10050",
			"main":        "1",
			"details":     []any{},
		}}),
	})
	iface, err := client.HostInterfaceGet(t.Context(), c, "42")
	if err != nil {
		t.Fatalf("HostInterfaceGet: %v", err)
	}
	if iface == nil {
		t.Fatal("expected non-nil interface")
	}
	if iface.InterfaceID != "42" {
		t.Errorf("InterfaceID = %q, want %q", iface.InterfaceID, "42")
	}
	if iface.HostID != "10084" {
		t.Errorf("HostID = %q, want %q", iface.HostID, "10084")
	}
	if iface.Type != 1 {
		t.Errorf("Type = %d, want 1", iface.Type)
	}
	if iface.UseIP != 1 {
		t.Errorf("UseIP = %d, want 1", iface.UseIP)
	}
	if iface.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want %q", iface.IP, "192.168.1.1")
	}
	if iface.Port != "10050" {
		t.Errorf("Port = %q, want %q", iface.Port, "10050")
	}
	if iface.Main != 1 {
		t.Errorf("Main = %d, want 1", iface.Main)
	}
	if iface.Details != nil {
		t.Errorf("Details = %+v, want nil for non-SNMP", iface.Details)
	}
}

func TestHostInterfaceGet_SNMPType(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{{
			"interfaceid": "7",
			"hostid":      "10084",
			"type":        "2",
			"useip":       "1",
			"ip":          "10.0.0.1",
			"dns":         "",
			"port":        "161",
			"main":        "1",
			"details": map[string]any{
				"version":        "2",
				"community":      "public",
				"bulk":           "1",
				"securityname":   "",
				"securitylevel":  "0",
				"authprotocol":   "0",
				"authpassphrase": "",
				"privprotocol":   "0",
				"privpassphrase": "",
				"contextname":    "",
			},
		}}),
	})
	iface, err := client.HostInterfaceGet(t.Context(), c, "7")
	if err != nil {
		t.Fatalf("HostInterfaceGet SNMP: %v", err)
	}
	if iface == nil {
		t.Fatal("expected non-nil interface")
	}
	if iface.Type != 2 {
		t.Errorf("Type = %d, want 2", iface.Type)
	}
	if iface.Details == nil {
		t.Fatal("expected non-nil Details for SNMP interface")
	}
	if iface.Details.Version != 2 {
		t.Errorf("Details.Version = %d, want 2", iface.Details.Version)
	}
	if iface.Details.Community != "public" {
		t.Errorf("Details.Community = %q, want %q", iface.Details.Community, "public")
	}
	if iface.Details.BulkRequests != 1 {
		t.Errorf("Details.BulkRequests = %d, want 1", iface.Details.BulkRequests)
	}
}

func TestHostInterfaceGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{}),
	})
	iface, err := client.HostInterfaceGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface != nil {
		t.Errorf("expected nil for not-found, got %+v", iface)
	}
}

func TestHostInterfaceGet_NonSNMPDetailsNormalized(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{{
			"interfaceid": "3",
			"hostid":      "10084",
			"type":        "3",
			"useip":       "0",
			"ip":          "",
			"dns":         "ipmi.example.com",
			"port":        "623",
			"main":        "1",
			"details":     []any{},
		}}),
	})
	iface, err := client.HostInterfaceGet(t.Context(), c, "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface == nil {
		t.Fatal("expected non-nil interface")
	}
	if iface.Details != nil {
		t.Errorf("Details should be nil for IPMI interface, got %+v", iface.Details)
	}
}

func TestHostInterfaceGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.HostInterfaceGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostInterfaceGetByHostAndType ----

func TestHostInterfaceGetByHostAndType_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{{
			"interfaceid": "1",
			"hostid":      "10084",
			"type":        "1",
			"useip":       "1",
			"ip":          "127.0.0.1",
			"dns":         "",
			"port":        "10050",
			"main":        "1",
			"details":     []any{},
		}}),
	})
	ifaces, err := client.HostInterfaceGetByHostAndType(t.Context(), c, "10084", 1)
	if err != nil {
		t.Fatalf("HostInterfaceGetByHostAndType: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("len(ifaces) = %d, want 1", len(ifaces))
	}
	if ifaces[0].HostID != "10084" {
		t.Errorf("HostID = %q, want %q", ifaces[0].HostID, "10084")
	}
	if ifaces[0].Type != 1 {
		t.Errorf("Type = %d, want 1", ifaces[0].Type)
	}
}

func TestHostInterfaceGetByHostAndType_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{}),
	})
	ifaces, err := client.HostInterfaceGetByHostAndType(t.Context(), c, "10084", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ifaces) != 0 {
		t.Errorf("expected empty slice, got %v", ifaces)
	}
}

func TestHostInterfaceGetByHostAndType_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcOK(t, []map[string]any{
			{
				"interfaceid": "1", "hostid": "10084", "type": "1",
				"useip": "1", "ip": "127.0.0.1", "dns": "", "port": "10050",
				"main": "1", "details": []any{},
			},
			{
				"interfaceid": "2", "hostid": "10084", "type": "1",
				"useip": "1", "ip": "10.0.0.1", "dns": "", "port": "10050",
				"main": "0", "details": []any{},
			},
		}),
	})
	ifaces, err := client.HostInterfaceGetByHostAndType(t.Context(), c, "10084", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ifaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(ifaces))
	}
}

func TestHostInterfaceGetByHostAndType_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.HostInterfaceGetByHostAndType(t.Context(), c, "1", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostInterfaceUpdate ----

func TestHostInterfaceUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.update": rpcOK(t, map[string]any{"interfaceids": []string{"42"}}),
	})
	iface := client.HostInterface{
		InterfaceID: "42",
		HostID:      "10084",
		Type:        1,
		UseIP:       1,
		IP:          "192.168.1.2",
		DNS:         "",
		Port:        "10050",
		Main:        1,
	}
	if err := client.HostInterfaceUpdate(t.Context(), c, iface); err != nil {
		t.Fatalf("HostInterfaceUpdate: %v", err)
	}
}

func TestHostInterfaceUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.update": rpcErr(t, -32602, "Invalid params."),
	})
	iface := client.HostInterface{InterfaceID: "1", Type: 1, Port: "10050"}
	if err := client.HostInterfaceUpdate(t.Context(), c, iface); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostInterfaceDelete ----

func TestHostInterfaceDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.delete": rpcOK(t, map[string]any{"interfaceids": []string{"42"}}),
	})
	if err := client.HostInterfaceDelete(t.Context(), c, "42"); err != nil {
		t.Fatalf("HostInterfaceDelete: %v", err)
	}
}

func TestHostInterfaceDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostinterface.delete": rpcErr(t, -32500, "Cannot delete interface."),
	})
	if err := client.HostInterfaceDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
