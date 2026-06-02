package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// methodDispatcher serves different responses based on the RPC method, after
// first handling the mandatory apiinfo.version call from client.New.
func methodDispatcher(t *testing.T, handlers map[string]http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		var req rpcMethod
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "apiinfo.version" {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": "7.0.0", "id": 1})
			return
		}
		if h, ok := handlers[req.Method]; ok {
			h(w, r)
			return
		}
		http.Error(w, "unexpected method: "+req.Method, http.StatusBadRequest)
	}
}

func rpcOK(t *testing.T, result any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": result, "id": 1})
	}
}

func rpcErr(t *testing.T, code int, message string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": code, "message": message, "data": ""},
			"id":      1,
		})
	}
}

func newTestClient(t *testing.T, handlers map[string]http.HandlerFunc) client.Client {
	t.Helper()
	srv := rpcServer(t, methodDispatcher(t, handlers))
	c, err := client.New(context.Background(), srv.URL, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// ---- HostGroupCreate ----

func TestHostGroupCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.create": rpcOK(t, map[string]any{"groupids": []string{"42"}}),
	})
	id, err := client.HostGroupCreate(t.Context(), c, "Linux servers")
	if err != nil {
		t.Fatalf("HostGroupCreate: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want %q", id, "42")
	}
}

func TestHostGroupCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.HostGroupCreate(t.Context(), c, "duplicate")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostGroupGet ----

func TestHostGroupGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcOK(t, []map[string]any{{"groupid": "5", "name": "Web servers"}}),
	})
	group, err := client.HostGroupGet(t.Context(), c, "5")
	if err != nil {
		t.Fatalf("HostGroupGet: %v", err)
	}
	if group == nil {
		t.Fatal("expected non-nil group")
	}
	if group.ID != "5" || group.Name != "Web servers" {
		t.Errorf("group = %+v, want {5, Web servers}", group)
	}
}

func TestHostGroupGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcOK(t, []map[string]any{}),
	})
	group, err := client.HostGroupGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group != nil {
		t.Errorf("expected nil for not-found, got %+v", group)
	}
}

func TestHostGroupGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.HostGroupGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostGroupGetByName ----

func TestHostGroupGetByName_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcOK(t, []map[string]any{{"groupid": "3", "name": "DB servers"}}),
	})
	groups, err := client.HostGroupGetByName(t.Context(), c, "DB servers")
	if err != nil {
		t.Fatalf("HostGroupGetByName: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Name != "DB servers" {
		t.Errorf("name = %q, want %q", groups[0].Name, "DB servers")
	}
}

func TestHostGroupGetByName_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcOK(t, []map[string]any{}),
	})
	groups, err := client.HostGroupGetByName(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty slice, got %v", groups)
	}
}

func TestHostGroupGetByName_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcOK(t, []map[string]any{
			{"groupid": "1", "name": "Servers"},
			{"groupid": "2", "name": "Servers"},
		}),
	})
	groups, err := client.HostGroupGetByName(t.Context(), c, "Servers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestHostGroupGetByName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.HostGroupGetByName(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostGroupUpdate ----

func TestHostGroupUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.update": rpcOK(t, map[string]any{"groupids": []string{"7"}}),
	})
	if err := client.HostGroupUpdate(t.Context(), c, "7", "Renamed group"); err != nil {
		t.Fatalf("HostGroupUpdate: %v", err)
	}
}

func TestHostGroupUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.update": rpcErr(t, -32602, "Invalid params."),
	})
	if err := client.HostGroupUpdate(t.Context(), c, "1", "x"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- HostGroupDelete ----

func TestHostGroupDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.delete": rpcOK(t, map[string]any{"groupids": []string{"9"}}),
	})
	if err := client.HostGroupDelete(t.Context(), c, "9"); err != nil {
		t.Fatalf("HostGroupDelete: %v", err)
	}
}

func TestHostGroupDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"hostgroup.delete": rpcErr(t, -32500, "Cannot delete host group."),
	})
	if err := client.HostGroupDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
