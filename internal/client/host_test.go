package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- HostCreate ----

func TestHostCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.create": rpcOK(t, map[string]any{"hostids": []string{"42"}}),
	})
	h := client.Host{
		Host:   "linux-srv-01",
		Name:   "Linux Server 01",
		Groups: []client.HostGroupRef{{GroupID: "5"}},
	}
	id, err := client.HostCreate(t.Context(), c, h)
	if err != nil {
		t.Fatalf("HostCreate: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want %q", id, "42")
	}
}

func TestHostCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.create": rpcErr(t, -32602, "Invalid params."),
	})
	h := client.Host{Host: "dup", Groups: []client.HostGroupRef{{GroupID: "1"}}}
	_, err := client.HostCreate(t.Context(), c, h)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostGet ----

func TestHostGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcOK(t, []map[string]any{{
			"hostid":         "42",
			"host":           "linux-srv-01",
			"name":           "Linux Server 01",
			"description":    "A test host",
			"status":         "0",
			"groups":         []map[string]any{{"groupid": "5"}},
			"tags":           []map[string]any{{"tag": "env", "value": "prod"}},
			"inventory":      map[string]any{"type": "1"},
			"inventory_mode": "0",
			"proxyid":        "0",
		}}),
	})
	h, err := client.HostGet(t.Context(), c, "42")
	if err != nil {
		t.Fatalf("HostGet: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil host")
	}
	if h.HostID != "42" {
		t.Errorf("HostID = %q, want %q", h.HostID, "42")
	}
	if h.Host != "linux-srv-01" {
		t.Errorf("Host = %q, want %q", h.Host, "linux-srv-01")
	}
	if h.Name != "Linux Server 01" {
		t.Errorf("Name = %q, want %q", h.Name, "Linux Server 01")
	}
	if h.Description != "A test host" {
		t.Errorf("Description = %q, want %q", h.Description, "A test host")
	}
	if h.Status != 0 {
		t.Errorf("Status = %d, want 0", h.Status)
	}
	if len(h.Groups) != 1 || h.Groups[0].GroupID != "5" {
		t.Errorf("Groups = %+v, want [{GroupID: 5}]", h.Groups)
	}
	if len(h.Tags) != 1 || h.Tags[0].Tag != "env" || h.Tags[0].Value != "prod" {
		t.Errorf("Tags = %+v, want [{env prod}]", h.Tags)
	}
	if h.InventoryMode != 0 {
		t.Errorf("InventoryMode = %d, want 0", h.InventoryMode)
	}
	if h.ProxyID != "0" {
		t.Errorf("ProxyID = %q, want %q", h.ProxyID, "0")
	}
}

func TestHostGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcOK(t, []map[string]any{}),
	})
	h, err := client.HostGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h != nil {
		t.Errorf("expected nil for not-found, got %+v", h)
	}
}

func TestHostGet_InventoryDisabled(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcOK(t, []map[string]any{{
			"hostid":         "1",
			"host":           "host01",
			"name":           "Host 01",
			"description":    "",
			"status":         "0",
			"groups":         []map[string]any{{"groupid": "5"}},
			"tags":           []map[string]any{},
			"inventory":      []any{},
			"inventory_mode": "-1",
			"proxyid":        "0",
		}}),
	})
	h, err := client.HostGet(t.Context(), c, "1")
	if err != nil {
		t.Fatalf("HostGet: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil host")
	}
	if h.InventoryMode != -1 {
		t.Errorf("InventoryMode = %d, want -1", h.InventoryMode)
	}
	if len(h.Inventory) != 0 {
		t.Errorf("Inventory = %v, want empty", h.Inventory)
	}
}

func TestHostGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.HostGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostGetByTechnicalName ----

func TestHostGetByTechnicalName_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcOK(t, []map[string]any{{
			"hostid":         "7",
			"host":           "db-srv-01",
			"name":           "DB Server 01",
			"description":    "",
			"status":         "0",
			"groups":         []map[string]any{{"groupid": "3"}},
			"tags":           []map[string]any{},
			"inventory":      []any{},
			"inventory_mode": "-1",
			"proxyid":        "0",
		}}),
	})
	hosts, err := client.HostGetByTechnicalName(t.Context(), c, "db-srv-01")
	if err != nil {
		t.Fatalf("HostGetByTechnicalName: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("len(hosts) = %d, want 1", len(hosts))
	}
	if hosts[0].Host != "db-srv-01" {
		t.Errorf("Host = %q, want %q", hosts[0].Host, "db-srv-01")
	}
}

func TestHostGetByTechnicalName_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcOK(t, []map[string]any{}),
	})
	hosts, err := client.HostGetByTechnicalName(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected empty slice, got %v", hosts)
	}
}

func TestHostGetByTechnicalName_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcOK(t, []map[string]any{
			{
				"hostid": "1", "host": "web-srv", "name": "Web Server",
				"description": "", "status": "0",
				"groups": []map[string]any{{"groupid": "1"}},
				"tags":   []map[string]any{}, "inventory": []any{},
				"inventory_mode": "-1", "proxyid": "0",
			},
			{
				"hostid": "2", "host": "web-srv", "name": "Web Server 2",
				"description": "", "status": "0",
				"groups": []map[string]any{{"groupid": "2"}},
				"tags":   []map[string]any{}, "inventory": []any{},
				"inventory_mode": "-1", "proxyid": "0",
			},
		}),
	})
	hosts, err := client.HostGetByTechnicalName(t.Context(), c, "web-srv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(hosts))
	}
}

func TestHostGetByTechnicalName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.HostGetByTechnicalName(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostUpdate ----

func TestHostUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.update": rpcOK(t, map[string]any{"hostids": []string{"7"}}),
	})
	h := client.Host{
		HostID: "7",
		Host:   "linux-srv-01",
		Groups: []client.HostGroupRef{{GroupID: "5"}},
	}
	if err := client.HostUpdate(t.Context(), c, h); err != nil {
		t.Fatalf("HostUpdate: %v", err)
	}
}

func TestHostUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.update": rpcErr(t, -32602, "Invalid params."),
	})
	h := client.Host{HostID: "1", Host: "x", Groups: []client.HostGroupRef{{GroupID: "1"}}}
	if err := client.HostUpdate(t.Context(), c, h); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostDelete ----

func TestHostDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.delete": rpcOK(t, map[string]any{"hostids": []string{"9"}}),
	})
	if err := client.HostDelete(t.Context(), c, "9"); err != nil {
		t.Fatalf("HostDelete: %v", err)
	}
}

func TestHostDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"host.delete": rpcErr(t, -32500, "Cannot delete host."),
	})
	if err := client.HostDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
