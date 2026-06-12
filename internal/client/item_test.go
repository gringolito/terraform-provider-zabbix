package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- ItemGet ----

func TestItemGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcOK(t, []map[string]any{{
			"itemid": "300",
			"key_":   "cpu.util",
			"name":   "CPU utilization",
			"hostid": "10084",
		}}),
	})
	item, err := client.ItemGet(t.Context(), c, "300")
	if err != nil {
		t.Fatalf("ItemGet: %v", err)
	}
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.ItemID != "300" {
		t.Errorf("ItemID = %q, want %q", item.ItemID, "300")
	}
	if item.Key != "cpu.util" {
		t.Errorf("Key = %q, want %q", item.Key, "cpu.util")
	}
	if item.Name != "CPU utilization" {
		t.Errorf("Name = %q, want %q", item.Name, "CPU utilization")
	}
	if item.HostID != "10084" {
		t.Errorf("HostID = %q, want %q", item.HostID, "10084")
	}
}

func TestItemGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcOK(t, []map[string]any{}),
	})
	item, err := client.ItemGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil for not-found, got %+v", item)
	}
}

func TestItemGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.ItemGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ItemGetByKeyAndScope ----

func TestItemGetByKeyAndScope_ByHostID(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcOK(t, []map[string]any{{
			"itemid": "301",
			"key_":   "vfs.fs.size[/,pfree]",
			"name":   "Free disk space",
			"hostid": "42",
		}}),
	})
	items, err := client.ItemGetByKeyAndScope(t.Context(), c, "vfs.fs.size[/,pfree]", "42", "")
	if err != nil {
		t.Fatalf("ItemGetByKeyAndScope: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].ItemID != "301" {
		t.Errorf("ItemID = %q, want %q", items[0].ItemID, "301")
	}
}

func TestItemGetByKeyAndScope_ByTemplateID(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcOK(t, []map[string]any{{
			"itemid": "302",
			"key_":   "cpu.util",
			"name":   "CPU utilization",
			"hostid": "10084",
		}}),
	})
	items, err := client.ItemGetByKeyAndScope(t.Context(), c, "cpu.util", "", "10084")
	if err != nil {
		t.Fatalf("ItemGetByKeyAndScope: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
}

func TestItemGetByKeyAndScope_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcOK(t, []map[string]any{}),
	})
	items, err := client.ItemGetByKeyAndScope(t.Context(), c, "not.found", "1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty, got %v", items)
	}
}

func TestItemGetByKeyAndScope_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"item.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.ItemGetByKeyAndScope(t.Context(), c, "cpu.util", "1", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
