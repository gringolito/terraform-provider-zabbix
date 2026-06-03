package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- MediaTypeCreate ----

func TestMediaTypeCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.create": rpcOK(t, map[string]any{"mediatypeids": []string{"10"}}),
	})
	id, err := client.MediaTypeCreate(t.Context(), c, client.MediaType{
		Name: "Email alerts",
		Type: client.MediaTypeTypeEmail,
	})
	if err != nil {
		t.Fatalf("MediaTypeCreate: %v", err)
	}
	if id != "10" {
		t.Errorf("id = %q, want %q", id, "10")
	}
}

func TestMediaTypeCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.MediaTypeCreate(t.Context(), c, client.MediaType{Name: "dup", Type: client.MediaTypeTypeEmail})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- MediaTypeGet ----

func TestMediaTypeGet_Success(t *testing.T) {
	// Zabbix 7.0 returns integer fields as JSON strings; verify the struct handles that.
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcOK(t, []map[string]any{{
			"mediatypeid": "5",
			"name":        "Email",
			"type":        "0",
			"status":      "0",
			"maxsessions": "1",
			"maxattempts": "3",
		}}),
	})
	mt, err := client.MediaTypeGet(t.Context(), c, "5")
	if err != nil {
		t.Fatalf("MediaTypeGet: %v", err)
	}
	if mt == nil {
		t.Fatal("expected non-nil media type")
	}
	if mt.ID != "5" || mt.Name != "Email" {
		t.Errorf("mt = %+v, want {ID:5, Name:Email}", mt)
	}
}

func TestMediaTypeGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcOK(t, []map[string]any{}),
	})
	mt, err := client.MediaTypeGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt != nil {
		t.Errorf("expected nil for not-found, got %+v", mt)
	}
}

func TestMediaTypeGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.MediaTypeGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- MediaTypeGetByName ----

func TestMediaTypeGetByName_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcOK(t, []map[string]any{{
			"mediatypeid": "3",
			"name":        "Webhook alerts",
			"type":        "4",
			"status":      "0",
			"maxsessions": "1",
			"maxattempts": "3",
		}}),
	})
	mts, err := client.MediaTypeGetByName(t.Context(), c, "Webhook alerts")
	if err != nil {
		t.Fatalf("MediaTypeGetByName: %v", err)
	}
	if len(mts) != 1 {
		t.Fatalf("len(mts) = %d, want 1", len(mts))
	}
	if mts[0].Name != "Webhook alerts" {
		t.Errorf("name = %q, want %q", mts[0].Name, "Webhook alerts")
	}
}

func TestMediaTypeGetByName_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcOK(t, []map[string]any{}),
	})
	mts, err := client.MediaTypeGetByName(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mts) != 0 {
		t.Errorf("expected empty slice, got %v", mts)
	}
}

func TestMediaTypeGetByName_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcOK(t, []map[string]any{
			{"mediatypeid": "1", "name": "Email", "type": "0", "status": "0", "maxsessions": "1", "maxattempts": "3"},
			{"mediatypeid": "2", "name": "Email", "type": "0", "status": "0", "maxsessions": "1", "maxattempts": "3"},
		}),
	})
	mts, err := client.MediaTypeGetByName(t.Context(), c, "Email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mts) != 2 {
		t.Errorf("expected 2 media types, got %d", len(mts))
	}
}

func TestMediaTypeGetByName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.MediaTypeGetByName(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- MediaTypeUpdate ----

func TestMediaTypeUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.update": rpcOK(t, map[string]any{"mediatypeids": []string{"7"}}),
	})
	if err := client.MediaTypeUpdate(t.Context(), c, client.MediaType{ID: "7", Name: "Updated", Type: client.MediaTypeTypeEmail}); err != nil {
		t.Fatalf("MediaTypeUpdate: %v", err)
	}
}

func TestMediaTypeUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.update": rpcErr(t, -32602, "Invalid params."),
	})
	if err := client.MediaTypeUpdate(t.Context(), c, client.MediaType{ID: "1", Name: "x", Type: client.MediaTypeTypeEmail}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- MediaTypeDelete ----

func TestMediaTypeDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.delete": rpcOK(t, map[string]any{"mediatypeids": []string{"9"}}),
	})
	if err := client.MediaTypeDelete(t.Context(), c, "9"); err != nil {
		t.Fatalf("MediaTypeDelete: %v", err)
	}
}

func TestMediaTypeDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"mediatype.delete": rpcErr(t, -32500, "Cannot delete media type."),
	})
	if err := client.MediaTypeDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
