package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- ActionCreate ----

func TestActionCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.create": rpcOK(t, map[string]any{"actionids": []string{"10"}}),
	})
	a := client.Action{
		Name:             "Notify admin",
		EscPeriod:        "1h",
		DefShortdata:     "Problem: {EVENT.NAME}",
		PauseSuppressed:  1,
		NotifyIfCanceled: 1,
		Filter:           client.ActionFilter{Conditions: []client.ActionCondition{}},
		Operations:       []client.ActionOperation{},
	}
	id, err := client.ActionCreate(t.Context(), c, a)
	if err != nil {
		t.Fatalf("ActionCreate: %v", err)
	}
	if id != "10" {
		t.Errorf("id = %q, want %q", id, "10")
	}
}

func TestActionCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.ActionCreate(t.Context(), c, client.Action{Name: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ActionGet ----

func TestActionGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{{
			"actionid":           "10",
			"name":               "Notify admin",
			"eventsource":        "0",
			"status":             "0",
			"esc_period":         "1h",
			"pause_suppressed":   "1",
			"notify_if_canceled": "1",
			"def_shortdata":      "Problem: {EVENT.NAME}",
			"def_longdata":       "",
			"filter": map[string]any{
				"evaltype":   "0",
				"formula":    "",
				"conditions": []any{},
			},
			"operations":          []any{},
			"recovery_operations": []any{},
			"update_operations":   []any{},
		}}),
	})
	a, err := client.ActionGet(t.Context(), c, "10")
	if err != nil {
		t.Fatalf("ActionGet: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil action")
	}
	if a.ActionID != "10" {
		t.Errorf("ActionID = %q, want %q", a.ActionID, "10")
	}
	if a.Name != "Notify admin" {
		t.Errorf("Name = %q, want %q", a.Name, "Notify admin")
	}
	if a.Status != 0 {
		t.Errorf("Status = %d, want 0", a.Status)
	}
	if a.EscPeriod != "1h" {
		t.Errorf("EscPeriod = %q, want %q", a.EscPeriod, "1h")
	}
	if a.PauseSuppressed != 1 {
		t.Errorf("PauseSuppressed = %d, want 1", a.PauseSuppressed)
	}
	if a.NotifyIfCanceled != 1 {
		t.Errorf("NotifyIfCanceled = %d, want 1", a.NotifyIfCanceled)
	}
	if a.DefShortdata != "Problem: {EVENT.NAME}" {
		t.Errorf("DefShortdata = %q, want %q", a.DefShortdata, "Problem: {EVENT.NAME}")
	}
	if a.Filter.EvalType != 0 {
		t.Errorf("Filter.EvalType = %d, want 0", a.Filter.EvalType)
	}
}

func TestActionGet_WithConditions(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{{
			"actionid":           "11",
			"name":               "Severity filter",
			"eventsource":        "0",
			"status":             "0",
			"esc_period":         "1h",
			"pause_suppressed":   "1",
			"notify_if_canceled": "1",
			"def_shortdata":      "",
			"def_longdata":       "",
			"filter": map[string]any{
				"evaltype": "0",
				"formula":  "",
				"conditions": []map[string]any{{
					"conditiontype": "4",
					"operator":      "5",
					"value":         "3",
					"value2":        "",
					"formulaid":     "A",
				}},
			},
			"operations":          []any{},
			"recovery_operations": []any{},
			"update_operations":   []any{},
		}}),
	})
	a, err := client.ActionGet(t.Context(), c, "11")
	if err != nil {
		t.Fatalf("ActionGet: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil action")
	}
	if len(a.Filter.Conditions) != 1 {
		t.Fatalf("len(Filter.Conditions) = %d, want 1", len(a.Filter.Conditions))
	}
	if a.Filter.Conditions[0].ConditionType != 4 {
		t.Errorf("ConditionType = %d, want 4", a.Filter.Conditions[0].ConditionType)
	}
	if a.Filter.Conditions[0].Operator != 5 {
		t.Errorf("Operator = %d, want 5", a.Filter.Conditions[0].Operator)
	}
	if a.Filter.Conditions[0].Value != "3" {
		t.Errorf("Value = %q, want %q", a.Filter.Conditions[0].Value, "3")
	}
	if a.Filter.Conditions[0].FormulaID != "A" {
		t.Errorf("FormulaID = %q, want %q", a.Filter.Conditions[0].FormulaID, "A")
	}
}

func TestActionGet_WithSendMessageOperation(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{{
			"actionid":           "12",
			"name":               "Send notification",
			"eventsource":        "0",
			"status":             "0",
			"esc_period":         "1h",
			"pause_suppressed":   "1",
			"notify_if_canceled": "1",
			"def_shortdata":      "",
			"def_longdata":       "",
			"filter": map[string]any{
				"evaltype":   "0",
				"formula":    "",
				"conditions": []any{},
			},
			"operations": []map[string]any{{
				"operationid":   "1",
				"operationtype": "0",
				"esc_period":    "0",
				"esc_step_from": "1",
				"esc_step_to":   "1",
				"opmessage": map[string]any{
					"operationid": "1",
					"default_msg": "1",
					"subject":     "",
					"message":     "",
					"mediatypeid": "0",
				},
				"opmessage_grp": []map[string]any{{"operationid": "1", "usrgrpid": "7"}},
				"opmessage_usr": []any{},
			}},
			"recovery_operations": []any{},
			"update_operations":   []any{},
		}}),
	})
	a, err := client.ActionGet(t.Context(), c, "12")
	if err != nil {
		t.Fatalf("ActionGet: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil action")
	}
	if len(a.Operations) != 1 {
		t.Fatalf("len(Operations) = %d, want 1", len(a.Operations))
	}
	op := a.Operations[0]
	if op.OperationType != 0 {
		t.Errorf("OperationType = %d, want 0", op.OperationType)
	}
	if op.EscStepFrom != 1 {
		t.Errorf("EscStepFrom = %d, want 1", op.EscStepFrom)
	}
	if op.EscStepTo != 1 {
		t.Errorf("EscStepTo = %d, want 1", op.EscStepTo)
	}
	if op.OpMessage == nil {
		t.Fatal("expected non-nil OpMessage")
	}
	if op.OpMessage.UseDefault != 1 {
		t.Errorf("OpMessage.UseDefault = %d, want 1", op.OpMessage.UseDefault)
	}
	if op.OpMessage.MediaTypeID != "0" {
		t.Errorf("OpMessage.MediaTypeID = %q, want %q", op.OpMessage.MediaTypeID, "0")
	}
	if len(op.OpMessageGrp) != 1 {
		t.Fatalf("len(OpMessageGrp) = %d, want 1", len(op.OpMessageGrp))
	}
	if op.OpMessageGrp[0].UserGroupID != "7" {
		t.Errorf("OpMessageGrp[0].UserGroupID = %q, want %q", op.OpMessageGrp[0].UserGroupID, "7")
	}
}

func TestActionGet_WithRemoteCommandOperation(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{{
			"actionid":           "13",
			"name":               "Remote command action",
			"eventsource":        "0",
			"status":             "0",
			"esc_period":         "1h",
			"pause_suppressed":   "1",
			"notify_if_canceled": "1",
			"def_shortdata":      "",
			"def_longdata":       "",
			"filter": map[string]any{
				"evaltype":   "0",
				"formula":    "",
				"conditions": []any{},
			},
			"operations": []map[string]any{{
				"operationid":   "2",
				"operationtype": "1",
				"esc_period":    "0",
				"esc_step_from": "1",
				"esc_step_to":   "1",
				"opcommand": map[string]any{
					"operationid": "2",
					"type":        "0",
					"scriptid":    "0",
					"execute_on":  "2",
					"port":        "",
					"authtype":    "0",
					"username":    "",
					"password":    "",
					"publickey":   "",
					"privatekey":  "",
					"command":     "echo hello",
				},
				"opcommand_hst": []map[string]any{{"operationid": "2", "hostid": "0"}},
				"opcommand_grp": []any{},
			}},
			"recovery_operations": []any{},
			"update_operations":   []any{},
		}}),
	})
	a, err := client.ActionGet(t.Context(), c, "13")
	if err != nil {
		t.Fatalf("ActionGet: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil action")
	}
	if len(a.Operations) != 1 {
		t.Fatalf("len(Operations) = %d, want 1", len(a.Operations))
	}
	op := a.Operations[0]
	if op.OperationType != 1 {
		t.Errorf("OperationType = %d, want 1", op.OperationType)
	}
	if op.OpCommand == nil {
		t.Fatal("expected non-nil OpCommand")
	}
	if op.OpCommand.Type != 0 {
		t.Errorf("OpCommand.Type = %d, want 0", op.OpCommand.Type)
	}
	if op.OpCommand.Command != "echo hello" {
		t.Errorf("OpCommand.Command = %q, want %q", op.OpCommand.Command, "echo hello")
	}
	if op.OpCommand.ExecuteOn != 2 {
		t.Errorf("OpCommand.ExecuteOn = %d, want 2", op.OpCommand.ExecuteOn)
	}
	if len(op.OpCommandHst) != 1 {
		t.Fatalf("len(OpCommandHst) = %d, want 1", len(op.OpCommandHst))
	}
	if op.OpCommandHst[0].HostID != "0" {
		t.Errorf("OpCommandHst[0].HostID = %q, want %q", op.OpCommandHst[0].HostID, "0")
	}
}

func TestActionGet_WithRecoveryOperation_NotifyAllInvolved(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{{
			"actionid":           "14",
			"name":               "Recovery notify all",
			"eventsource":        "0",
			"status":             "0",
			"esc_period":         "1h",
			"pause_suppressed":   "1",
			"notify_if_canceled": "1",
			"def_shortdata":      "",
			"def_longdata":       "",
			"filter": map[string]any{
				"evaltype":   "0",
				"formula":    "",
				"conditions": []any{},
			},
			"operations": []any{},
			"recovery_operations": []map[string]any{{
				"operationid":   "3",
				"operationtype": "11",
				"opmessage_grp": []any{},
				"opmessage_usr": []any{},
			}},
			"update_operations": []any{},
		}}),
	})
	a, err := client.ActionGet(t.Context(), c, "14")
	if err != nil {
		t.Fatalf("ActionGet: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil action")
	}
	if len(a.RecoveryOperations) != 1 {
		t.Fatalf("len(RecoveryOperations) = %d, want 1", len(a.RecoveryOperations))
	}
	if a.RecoveryOperations[0].OperationType != 11 {
		t.Errorf("RecoveryOperations[0].OperationType = %d, want 11", a.RecoveryOperations[0].OperationType)
	}
}

func TestActionGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{}),
	})
	a, err := client.ActionGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil for not-found, got %+v", a)
	}
}

func TestActionGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.ActionGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ActionGetByName ----

func TestActionGetByName_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{{
			"actionid":           "20",
			"name":               "My Action",
			"eventsource":        "0",
			"status":             "0",
			"esc_period":         "1h",
			"pause_suppressed":   "1",
			"notify_if_canceled": "1",
			"def_shortdata":      "",
			"def_longdata":       "",
			"filter": map[string]any{
				"evaltype":   "0",
				"formula":    "",
				"conditions": []any{},
			},
			"operations":          []any{},
			"recovery_operations": []any{},
			"update_operations":   []any{},
		}}),
	})
	actions, err := client.ActionGetByName(t.Context(), c, "My Action")
	if err != nil {
		t.Fatalf("ActionGetByName: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("len = %d, want 1", len(actions))
	}
	if actions[0].ActionID != "20" {
		t.Errorf("ActionID = %q, want %q", actions[0].ActionID, "20")
	}
}

func TestActionGetByName_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{}),
	})
	actions, err := client.ActionGetByName(t.Context(), c, "Nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected empty, got %v", actions)
	}
}

func TestActionGetByName_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcOK(t, []map[string]any{
			{
				"actionid": "21", "name": "Action A", "eventsource": "0", "status": "0",
				"esc_period": "1h", "pause_suppressed": "1", "notify_if_canceled": "1",
				"def_shortdata": "", "def_longdata": "",
				"filter":              map[string]any{"evaltype": "0", "formula": "", "conditions": []any{}},
				"operations":          []any{},
				"recovery_operations": []any{},
				"update_operations":   []any{},
			},
			{
				"actionid": "22", "name": "Action B", "eventsource": "0", "status": "0",
				"esc_period": "1h", "pause_suppressed": "1", "notify_if_canceled": "1",
				"def_shortdata": "", "def_longdata": "",
				"filter":              map[string]any{"evaltype": "0", "formula": "", "conditions": []any{}},
				"operations":          []any{},
				"recovery_operations": []any{},
				"update_operations":   []any{},
			},
		}),
	})
	actions, err := client.ActionGetByName(t.Context(), c, "Action")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("len = %d, want 2", len(actions))
	}
}

func TestActionGetByName_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.ActionGetByName(t.Context(), c, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ActionUpdate ----

func TestActionUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.update": rpcOK(t, map[string]any{"actionids": []string{"10"}}),
	})
	a := client.Action{
		ActionID: "10",
		Name:     "Updated action",
	}
	if err := client.ActionUpdate(t.Context(), c, a); err != nil {
		t.Fatalf("ActionUpdate: %v", err)
	}
}

func TestActionUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.update": rpcErr(t, -32602, "Invalid params."),
	})
	a := client.Action{ActionID: "1", Name: "test"}
	if err := client.ActionUpdate(t.Context(), c, a); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- ActionDelete ----

func TestActionDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.delete": rpcOK(t, map[string]any{"actionids": []string{"10"}}),
	})
	if err := client.ActionDelete(t.Context(), c, "10"); err != nil {
		t.Fatalf("ActionDelete: %v", err)
	}
}

func TestActionDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"action.delete": rpcErr(t, -32500, "Cannot delete action."),
	})
	if err := client.ActionDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
