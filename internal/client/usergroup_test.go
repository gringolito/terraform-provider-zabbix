package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- UserGroupCreate ----

func TestUserGroupCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.create": rpcOK(t, map[string]any{"usrgrpids": []string{"42"}}),
	})
	ug := client.UserGroup{Name: "Network administrators", GUIAccess: 1, DebugMode: 0, UsersStatus: 0}
	id, err := client.UserGroupCreate(t.Context(), c, ug)
	if err != nil {
		t.Fatalf("UserGroupCreate: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, want %q", id, "42")
	}
}

func TestUserGroupCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.UserGroupCreate(t.Context(), c, client.UserGroup{Name: "duplicate"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- UserGroupGet ----

func TestUserGroupGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcOK(t, []map[string]any{{
			"usrgrpid":     "5",
			"name":         "Network administrators",
			"gui_access":   "1",
			"debug_mode":   "0",
			"users_status": "0",
		}}),
	})
	ug, err := client.UserGroupGet(t.Context(), c, "5")
	if err != nil {
		t.Fatalf("UserGroupGet: %v", err)
	}
	if ug == nil {
		t.Fatal("expected non-nil user group")
	}
	if ug.ID != "5" || ug.Name != "Network administrators" {
		t.Errorf("id/name = %q/%q, want 5/Network administrators", ug.ID, ug.Name)
	}
	if ug.GUIAccess != 1 {
		t.Errorf("GUIAccess = %d, want 1", ug.GUIAccess)
	}
	if ug.DebugMode != 0 {
		t.Errorf("DebugMode = %d, want 0", ug.DebugMode)
	}
	if ug.UsersStatus != 0 {
		t.Errorf("UsersStatus = %d, want 0", ug.UsersStatus)
	}
}

func TestUserGroupGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcOK(t, []map[string]any{}),
	})
	ug, err := client.UserGroupGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ug != nil {
		t.Errorf("expected nil for not-found, got %+v", ug)
	}
}

func TestUserGroupGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.UserGroupGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- UserGroupGetByName ----

func TestUserGroupGetByName_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcOK(t, []map[string]any{{
			"usrgrpid":     "3",
			"name":         "Network administrators",
			"gui_access":   "0",
			"debug_mode":   "0",
			"users_status": "0",
		}}),
	})
	groups, err := client.UserGroupGetByName(t.Context(), c, "Network administrators")
	if err != nil {
		t.Fatalf("UserGroupGetByName: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Name != "Network administrators" {
		t.Errorf("name = %q, want %q", groups[0].Name, "Network administrators")
	}
}

func TestUserGroupGetByName_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcOK(t, []map[string]any{}),
	})
	groups, err := client.UserGroupGetByName(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty slice, got %v", groups)
	}
}

func TestUserGroupGetByName_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcOK(t, []map[string]any{
			{"usrgrpid": "1", "name": "Admins", "gui_access": "0", "debug_mode": "0", "users_status": "0"},
			{"usrgrpid": "2", "name": "Admins", "gui_access": "0", "debug_mode": "0", "users_status": "0"},
		}),
	})
	groups, err := client.UserGroupGetByName(t.Context(), c, "Admins")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestUserGroupGetByName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.UserGroupGetByName(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- UserGroupUpdate ----

func TestUserGroupUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.update": rpcOK(t, map[string]any{"usrgrpids": []string{"7"}}),
	})
	ug := client.UserGroup{ID: "7", Name: "Renamed group", GUIAccess: 0, DebugMode: 1, UsersStatus: 0}
	if err := client.UserGroupUpdate(t.Context(), c, ug); err != nil {
		t.Fatalf("UserGroupUpdate: %v", err)
	}
}

func TestUserGroupUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.update": rpcErr(t, -32602, "Invalid params."),
	})
	if err := client.UserGroupUpdate(t.Context(), c, client.UserGroup{ID: "1", Name: "x"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- UserGroupDelete ----

func TestUserGroupDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.delete": rpcOK(t, map[string]any{"usrgrpids": []string{"9"}}),
	})
	if err := client.UserGroupDelete(t.Context(), c, "9"); err != nil {
		t.Fatalf("UserGroupDelete: %v", err)
	}
}

func TestUserGroupDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"usergroup.delete": rpcErr(t, -32500, "Cannot delete user group."),
	})
	if err := client.UserGroupDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
