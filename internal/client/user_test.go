package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

var testUserResponse = map[string]any{
	"userid":         "1",
	"username":       "Admin",
	"name":           "Zabbix",
	"surname":        "Administrator",
	"url":            "",
	"autologin":      "0",
	"autologout":     "0",
	"lang":           "default",
	"refresh":        "30s",
	"theme":          "default",
	"attempt_failed": "0",
	"attempt_ip":     "",
	"attempt_clock":  "0",
	"timezone":       "default",
	"roleid":         "3",
	"provisioned":    "0",
	"gui_access":     "0",
	"debug_mode":     "0",
	"users_status":   "0",
}

// ---- UserGet ----

func TestUserGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"user.get": rpcOK(t, []map[string]any{testUserResponse}),
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
	if user.URL != "" {
		t.Errorf("url = %q, want %q", user.URL, "")
	}
	if user.AutoLogin != "0" {
		t.Errorf("autologin = %q, want %q", user.AutoLogin, "0")
	}
	if user.AutoLogout != "0" {
		t.Errorf("autologout = %q, want %q", user.AutoLogout, "0")
	}
	if user.Language != "default" {
		t.Errorf("lang = %q, want %q", user.Language, "default")
	}
	if user.Refresh != "30s" {
		t.Errorf("refresh = %q, want %q", user.Refresh, "30s")
	}
	if user.Theme != "default" {
		t.Errorf("theme = %q, want %q", user.Theme, "default")
	}
	if user.AttemptFailed != "0" {
		t.Errorf("attempt_failed = %q, want %q", user.AttemptFailed, "0")
	}
	if user.AttemptIP != "" {
		t.Errorf("attempt_ip = %q, want %q", user.AttemptIP, "")
	}
	if user.AttemptClock != "0" {
		t.Errorf("attempt_clock = %q, want %q", user.AttemptClock, "0")
	}
	if user.Timezone != "default" {
		t.Errorf("timezone = %q, want %q", user.Timezone, "default")
	}
	if user.Provisioned != "0" {
		t.Errorf("provisioned = %q, want %q", user.Provisioned, "0")
	}
	if user.GUIAccess != 0 {
		t.Errorf("gui_access = %d, want 0", user.GUIAccess)
	}
	if user.DebugMode != 0 {
		t.Errorf("debug_mode = %d, want 0", user.DebugMode)
	}
	if user.UsersStatus != 0 {
		t.Errorf("users_status = %d, want 0", user.UsersStatus)
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
		"user.get": rpcOK(t, []map[string]any{testUserResponse}),
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
	if users[0].Language != "default" {
		t.Errorf("lang = %q, want %q", users[0].Language, "default")
	}
	if users[0].GUIAccess != 0 {
		t.Errorf("gui_access = %d, want 0", users[0].GUIAccess)
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
