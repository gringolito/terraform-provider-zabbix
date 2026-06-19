package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- UserGet ----

func TestUserGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcOK(t, []map[string]any{{
			"userid":   "1",
			"username": "Admin",
			"name":     "Zabbix",
			"surname":  "Administrator",
			"type":     "3",
			"roleid":   "3",
		}}),
	})
	user, err := client.UserGet(t.Context(), c, "1")
	if err != nil {
		t.Fatalf("UserGet: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.UserID != "1" || user.Username != "Admin" {
		t.Errorf("userid/username = %q/%q, want 1/Admin", user.UserID, user.Username)
	}
	if user.Name != "Zabbix" {
		t.Errorf("name = %q, want %q", user.Name, "Zabbix")
	}
	if user.Surname != "Administrator" {
		t.Errorf("surname = %q, want %q", user.Surname, "Administrator")
	}
	if user.Type != 3 {
		t.Errorf("type = %d, want 3", user.Type)
	}
	if user.RoleID != "3" {
		t.Errorf("roleid = %q, want %q", user.RoleID, "3")
	}
}

func TestUserGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcOK(t, []map[string]any{}),
	})
	user, err := client.UserGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for not-found, got %+v", user)
	}
}

func TestUserGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.UserGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- UserGetByUsername ----

func TestUserGetByUsername_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcOK(t, []map[string]any{{
			"userid":   "1",
			"username": "Admin",
			"name":     "Zabbix",
			"surname":  "Administrator",
			"type":     "3",
			"roleid":   "3",
		}}),
	})
	users, err := client.UserGetByUsername(t.Context(), c, "Admin")
	if err != nil {
		t.Fatalf("UserGetByUsername: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("len(users) = %d, want 1", len(users))
	}
	if users[0].Username != "Admin" {
		t.Errorf("username = %q, want %q", users[0].Username, "Admin")
	}
}

func TestUserGetByUsername_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcOK(t, []map[string]any{}),
	})
	users, err := client.UserGetByUsername(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected empty slice, got %v", users)
	}
}

func TestUserGetByUsername_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.UserGetByUsername(t.Context(), c, "Admin")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
