package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- RoleCreate ----

func TestRoleCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.create": rpcOK(t, map[string]any{"roleids": []string{"5"}}),
	})
	role := client.Role{Name: "Custom role", Type: 1}
	id, err := client.RoleCreate(t.Context(), c, role)
	if err != nil {
		t.Fatalf("RoleCreate: %v", err)
	}
	if id != "5" {
		t.Errorf("id = %q, want %q", id, "5")
	}
}

func TestRoleCreate_WithRules_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.create": rpcOK(t, map[string]any{"roleids": []string{"5"}}),
	})
	role := client.Role{
		Name:     "Custom role",
		Type:     1,
		HasRules: true,
		Rules: client.RoleRules{
			UIDefaultAccess:      1,
			ModulesDefaultAccess: 1,
			ActionsDefaultAccess: 1,
			APIAccess:            1,
			APIMode:              0,
		},
	}
	id, err := client.RoleCreate(t.Context(), c, role)
	if err != nil {
		t.Fatalf("RoleCreate: %v", err)
	}
	if id != "5" {
		t.Errorf("id = %q, want %q", id, "5")
	}
}

func TestRoleCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.RoleCreate(t.Context(), c, client.Role{Name: "duplicate"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- RoleGet ----

func TestRoleGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcOK(t, []map[string]any{{
			"roleid":   "5",
			"name":     "Custom role",
			"type":     "1",
			"readonly": "0",
			"rules": map[string]any{
				"ui":                     []any{},
				"ui.default_access":      "1",
				"modules.default_access": "1",
				"actions.default_access": "1",
				"api.access":             "1",
				"api.mode":               "0",
				"api":                    []any{},
				"modules":                []any{},
				"actions":                []any{},
			},
		}}),
	})
	role, err := client.RoleGet(t.Context(), c, "5")
	if err != nil {
		t.Fatalf("RoleGet: %v", err)
	}
	if role == nil {
		t.Fatal("expected non-nil role")
	}
	if role.ID != "5" || role.Name != "Custom role" {
		t.Errorf("id/name = %q/%q, want 5/Custom role", role.ID, role.Name)
	}
	if role.Type != 1 {
		t.Errorf("Type = %d, want 1", role.Type)
	}
	if role.ReadOnly != 0 {
		t.Errorf("ReadOnly = %d, want 0", role.ReadOnly)
	}
	if role.Rules.UIDefaultAccess != 1 {
		t.Errorf("UIDefaultAccess = %d, want 1", role.Rules.UIDefaultAccess)
	}
	if role.Rules.ModulesDefaultAccess != 1 {
		t.Errorf("ModulesDefaultAccess = %d, want 1", role.Rules.ModulesDefaultAccess)
	}
	if role.Rules.ActionsDefaultAccess != 1 {
		t.Errorf("ActionsDefaultAccess = %d, want 1", role.Rules.ActionsDefaultAccess)
	}
}

func TestRoleGet_WithRules(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcOK(t, []map[string]any{{
			"roleid":   "7",
			"name":     "Power user",
			"type":     "2",
			"readonly": "0",
			"rules": map[string]any{
				"ui": []any{
					map[string]any{"name": "monitoring.dashboard", "status": "1"},
					map[string]any{"name": "monitoring.problems", "status": "0"},
				},
				"ui.default_access":      "1",
				"modules.default_access": "1",
				"actions.default_access": "1",
				"api.access":             "1",
				"api.mode":               "1",
				"api":                    []any{"host.get", "item.get"},
				"modules":                []any{map[string]any{"moduleid": 3, "status": "1"}},
				"actions":                []any{map[string]any{"name": "edit_maps", "status": "1"}},
			},
		}}),
	})
	role, err := client.RoleGet(t.Context(), c, "7")
	if err != nil {
		t.Fatalf("RoleGet: %v", err)
	}
	if role == nil {
		t.Fatal("expected non-nil role")
	}
	if len(role.Rules.UI) != 2 {
		t.Fatalf("len(Rules.UI) = %d, want 2", len(role.Rules.UI))
	}
	if role.Rules.UI[0].Name != "monitoring.dashboard" || role.Rules.UI[0].Status != 1 {
		t.Errorf("UI[0] = %+v, want {Name:monitoring.dashboard Status:1}", role.Rules.UI[0])
	}
	if role.Rules.UIDefaultAccess != 1 {
		t.Errorf("UIDefaultAccess = %d, want 1", role.Rules.UIDefaultAccess)
	}
	if role.Rules.APIAccess != 1 {
		t.Errorf("APIAccess = %d, want 1", role.Rules.APIAccess)
	}
	if role.Rules.APIMode != 1 {
		t.Errorf("APIMode = %d, want 1", role.Rules.APIMode)
	}
	if len(role.Rules.APIMethods) != 2 {
		t.Fatalf("len(APIMethods) = %d, want 2", len(role.Rules.APIMethods))
	}
	if len(role.Rules.Modules) != 1 {
		t.Fatalf("len(Modules) = %d, want 1", len(role.Rules.Modules))
	}
	if role.Rules.Modules[0].ModuleID != 3 || role.Rules.Modules[0].Status != 1 {
		t.Errorf("Modules[0] = %+v, want {ModuleID:3 Status:1}", role.Rules.Modules[0])
	}
	if len(role.Rules.Actions) != 1 {
		t.Fatalf("len(Actions) = %d, want 1", len(role.Rules.Actions))
	}
	if role.Rules.Actions[0].Name != "edit_maps" || role.Rules.Actions[0].Status != 1 {
		t.Errorf("Actions[0] = %+v, want {Name:edit_maps Status:1}", role.Rules.Actions[0])
	}
}

func TestRoleGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcOK(t, []map[string]any{}),
	})
	role, err := client.RoleGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != nil {
		t.Errorf("expected nil for not-found, got %+v", role)
	}
}

func TestRoleGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.RoleGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- RoleGetByName ----

func TestRoleGetByName_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcOK(t, []map[string]any{{
			"roleid":   "3",
			"name":     "Custom role",
			"type":     "1",
			"readonly": "0",
			"rules": map[string]any{
				"ui":                     []any{},
				"ui.default_access":      "1",
				"modules.default_access": "1",
				"actions.default_access": "1",
				"api.access":             "1",
				"api.mode":               "0",
				"api":                    []any{},
				"modules":                []any{},
				"actions":                []any{},
			},
		}}),
	})
	roles, err := client.RoleGetByName(t.Context(), c, "Custom role")
	if err != nil {
		t.Fatalf("RoleGetByName: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("len(roles) = %d, want 1", len(roles))
	}
	if roles[0].Name != "Custom role" {
		t.Errorf("name = %q, want %q", roles[0].Name, "Custom role")
	}
}

func TestRoleGetByName_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcOK(t, []map[string]any{}),
	})
	roles, err := client.RoleGetByName(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("expected empty slice, got %v", roles)
	}
}

func TestRoleGetByName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.RoleGetByName(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- RoleUpdate ----

func TestRoleUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.update": rpcOK(t, map[string]any{"roleids": []string{"5"}}),
	})
	role := client.Role{ID: "5", Name: "Renamed role", Type: 1}
	if err := client.RoleUpdate(t.Context(), c, role); err != nil {
		t.Fatalf("RoleUpdate: %v", err)
	}
}

func TestRoleUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.update": rpcErr(t, -32602, "Invalid params."),
	})
	if err := client.RoleUpdate(t.Context(), c, client.Role{ID: "1", Name: "x"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- RoleDelete ----

func TestRoleDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.delete": rpcOK(t, map[string]any{"roleids": []string{"9"}}),
	})
	if err := client.RoleDelete(t.Context(), c, "9"); err != nil {
		t.Fatalf("RoleDelete: %v", err)
	}
}

func TestRoleDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"role.delete": rpcErr(t, -32500, "Cannot delete role."),
	})
	if err := client.RoleDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
