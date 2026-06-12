package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- TriggerCreate ----

func TestTriggerCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.create": rpcOK(t, map[string]any{"triggerids": []string{"100"}}),
	})
	tr := client.Trigger{
		Description: "High CPU usage",
		Expression:  `last(/web01/cpu.util)>90`,
		Priority:    4,
	}
	id, err := client.TriggerCreate(t.Context(), c, tr)
	if err != nil {
		t.Fatalf("TriggerCreate: %v", err)
	}
	if id != "100" {
		t.Errorf("id = %q, want %q", id, "100")
	}
}

func TestTriggerCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.create": rpcErr(t, -32602, "Invalid params."),
	})
	tr := client.Trigger{Description: "test", Expression: `last(/h/k)>0`}
	_, err := client.TriggerCreate(t.Context(), c, tr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TriggerGet ----

func TestTriggerGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcOK(t, []map[string]any{{
			"triggerid":           "100",
			"description":         "High CPU usage",
			"expression":          `{web01:cpu.util.last(0)}>90`,
			"recovery_mode":       "0",
			"recovery_expression": "",
			"priority":            "4",
			"status":              "0",
			"manual_close":        "0",
			"comments":            "",
			"url":                 "",
			"tags":                []any{},
		}}),
	})
	tr, err := client.TriggerGet(t.Context(), c, "100")
	if err != nil {
		t.Fatalf("TriggerGet: %v", err)
	}
	if tr == nil {
		t.Fatal("expected non-nil trigger")
	}
	if tr.TriggerID != "100" {
		t.Errorf("TriggerID = %q, want %q", tr.TriggerID, "100")
	}
	if tr.Description != "High CPU usage" {
		t.Errorf("Description = %q, want %q", tr.Description, "High CPU usage")
	}
	if tr.Priority != 4 {
		t.Errorf("Priority = %d, want 4", tr.Priority)
	}
	if tr.Status != 0 {
		t.Errorf("Status = %d, want 0", tr.Status)
	}
}

func TestTriggerGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcOK(t, []map[string]any{}),
	})
	tr, err := client.TriggerGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr != nil {
		t.Errorf("expected nil for not-found, got %+v", tr)
	}
}

func TestTriggerGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.TriggerGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTriggerGet_WithTags(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcOK(t, []map[string]any{{
			"triggerid":           "200",
			"description":         "Disk full",
			"expression":          `last(/h/vfs.fs.size[/,pfree])<10`,
			"recovery_mode":       "0",
			"recovery_expression": "",
			"priority":            "3",
			"status":              "0",
			"manual_close":        "0",
			"comments":            "Check disk usage",
			"url":                 "",
			"tags": []map[string]any{
				{"tag": "env", "value": "prod"},
				{"tag": "team", "value": "ops"},
			},
		}}),
	})
	tr, err := client.TriggerGet(t.Context(), c, "200")
	if err != nil {
		t.Fatalf("TriggerGet: %v", err)
	}
	if tr == nil {
		t.Fatal("expected non-nil trigger")
	}
	if len(tr.Tags) != 2 {
		t.Fatalf("len(Tags) = %d, want 2", len(tr.Tags))
	}
	if tr.Tags[0].Tag != "env" {
		t.Errorf("Tags[0].Tag = %q, want %q", tr.Tags[0].Tag, "env")
	}
	if tr.Tags[0].Value != "prod" {
		t.Errorf("Tags[0].Value = %q, want %q", tr.Tags[0].Value, "prod")
	}
	if tr.Comments != "Check disk usage" {
		t.Errorf("Comments = %q, want %q", tr.Comments, "Check disk usage")
	}
}

// ---- TriggerGetByDescriptionAndScope ----

func TestTriggerGetByDescriptionAndScope_ByHostID(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcOK(t, []map[string]any{{
			"triggerid":           "50",
			"description":         "CPU high",
			"expression":          `last(/h/cpu.util)>90`,
			"recovery_mode":       "0",
			"recovery_expression": "",
			"priority":            "2",
			"status":              "0",
			"manual_close":        "0",
			"comments":            "",
			"url":                 "",
			"tags":                []any{},
		}}),
	})
	trs, err := client.TriggerGetByDescriptionAndScope(t.Context(), c, "CPU high", "42", "")
	if err != nil {
		t.Fatalf("TriggerGetByDescriptionAndScope: %v", err)
	}
	if len(trs) != 1 {
		t.Fatalf("len = %d, want 1", len(trs))
	}
	if trs[0].TriggerID != "50" {
		t.Errorf("TriggerID = %q, want %q", trs[0].TriggerID, "50")
	}
}

func TestTriggerGetByDescriptionAndScope_ByTemplateID(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcOK(t, []map[string]any{{
			"triggerid":           "51",
			"description":         "Mem high",
			"expression":          `last(/tmpl/vm.memory.size[available])<100M`,
			"recovery_mode":       "0",
			"recovery_expression": "",
			"priority":            "2",
			"status":              "0",
			"manual_close":        "0",
			"comments":            "",
			"url":                 "",
			"tags":                []any{},
		}}),
	})
	trs, err := client.TriggerGetByDescriptionAndScope(t.Context(), c, "Mem high", "", "10084")
	if err != nil {
		t.Fatalf("TriggerGetByDescriptionAndScope: %v", err)
	}
	if len(trs) != 1 {
		t.Fatalf("len = %d, want 1", len(trs))
	}
}

func TestTriggerGetByDescriptionAndScope_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcOK(t, []map[string]any{}),
	})
	trs, err := client.TriggerGetByDescriptionAndScope(t.Context(), c, "Not found", "1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trs) != 0 {
		t.Errorf("expected empty, got %v", trs)
	}
}

func TestTriggerGetByDescriptionAndScope_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.TriggerGetByDescriptionAndScope(t.Context(), c, "test", "1", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TriggerUpdate ----

func TestTriggerUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.update": rpcOK(t, map[string]any{"triggerids": []string{"100"}}),
	})
	tr := client.Trigger{
		TriggerID:   "100",
		Description: "High CPU usage updated",
		Expression:  `last(/web01/cpu.util)>95`,
		Priority:    3,
	}
	if err := client.TriggerUpdate(t.Context(), c, tr); err != nil {
		t.Fatalf("TriggerUpdate: %v", err)
	}
}

func TestTriggerUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.update": rpcErr(t, -32602, "Invalid params."),
	})
	tr := client.Trigger{TriggerID: "1", Description: "test", Expression: `last(/h/k)>0`}
	if err := client.TriggerUpdate(t.Context(), c, tr); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TriggerDelete ----

func TestTriggerDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.delete": rpcOK(t, map[string]any{"triggerids": []string{"100"}}),
	})
	if err := client.TriggerDelete(t.Context(), c, "100"); err != nil {
		t.Fatalf("TriggerDelete: %v", err)
	}
}

func TestTriggerDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"trigger.delete": rpcErr(t, -32500, "Cannot delete trigger."),
	})
	if err := client.TriggerDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
