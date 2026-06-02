package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- TemplateGroupCreate ----

func TestTemplateGroupCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.create": rpcOK(t, map[string]any{"groupids": []string{"42"}}),
	})
	id, err := client.TemplateGroupCreate(t.Context(), c, "Linux templates")
	if err != nil {
		t.Fatalf("TemplateGroupCreate: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want %q", id, "42")
	}
}

func TestTemplateGroupCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.TemplateGroupCreate(t.Context(), c, "duplicate")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateGroupGet ----

func TestTemplateGroupGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcOK(t, []map[string]any{{"groupid": "5", "name": "Web templates"}}),
	})
	group, err := client.TemplateGroupGet(t.Context(), c, "5")
	if err != nil {
		t.Fatalf("TemplateGroupGet: %v", err)
	}
	if group == nil {
		t.Fatal("expected non-nil group")
	}
	if group.ID != "5" || group.Name != "Web templates" {
		t.Errorf("group = %+v, want {5, Web templates}", group)
	}
}

func TestTemplateGroupGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcOK(t, []map[string]any{}),
	})
	group, err := client.TemplateGroupGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group != nil {
		t.Errorf("expected nil for not-found, got %+v", group)
	}
}

func TestTemplateGroupGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.TemplateGroupGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateGroupGetByName ----

func TestTemplateGroupGetByName_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcOK(t, []map[string]any{{"groupid": "3", "name": "DB templates"}}),
	})
	groups, err := client.TemplateGroupGetByName(t.Context(), c, "DB templates")
	if err != nil {
		t.Fatalf("TemplateGroupGetByName: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Name != "DB templates" {
		t.Errorf("name = %q, want %q", groups[0].Name, "DB templates")
	}
}

func TestTemplateGroupGetByName_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcOK(t, []map[string]any{}),
	})
	groups, err := client.TemplateGroupGetByName(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty slice, got %v", groups)
	}
}

func TestTemplateGroupGetByName_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcOK(t, []map[string]any{
			{"groupid": "1", "name": "Templates"},
			{"groupid": "2", "name": "Templates"},
		}),
	})
	groups, err := client.TemplateGroupGetByName(t.Context(), c, "Templates")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestTemplateGroupGetByName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.TemplateGroupGetByName(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateGroupUpdate ----

func TestTemplateGroupUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.update": rpcOK(t, map[string]any{"groupids": []string{"7"}}),
	})
	if err := client.TemplateGroupUpdate(t.Context(), c, "7", "Renamed template group"); err != nil {
		t.Fatalf("TemplateGroupUpdate: %v", err)
	}
}

func TestTemplateGroupUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.update": rpcErr(t, -32602, "Invalid params."),
	})
	if err := client.TemplateGroupUpdate(t.Context(), c, "1", "x"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateGroupDelete ----

func TestTemplateGroupDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.delete": rpcOK(t, map[string]any{"groupids": []string{"9"}}),
	})
	if err := client.TemplateGroupDelete(t.Context(), c, "9"); err != nil {
		t.Fatalf("TemplateGroupDelete: %v", err)
	}
}

func TestTemplateGroupDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"templategroup.delete": rpcErr(t, -32500, "Cannot delete template group."),
	})
	if err := client.TemplateGroupDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
