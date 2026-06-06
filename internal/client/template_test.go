package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// ---- TemplateCreate ----

func TestTemplateCreate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.create": rpcOK(t, map[string]any{"templateids": []string{"10"}}),
	})
	id, err := client.TemplateCreate(t.Context(), c, client.Template{
		Host:        "Linux by Zabbix agent",
		Name:        "Linux by Zabbix agent",
		Description: "",
		Groups:      []client.TemplateGroupRef{{GroupID: "1"}},
	})
	if err != nil {
		t.Fatalf("TemplateCreate: %v", err)
	}
	if id != "10" {
		t.Errorf("id = %q, want %q", id, "10")
	}
}

func TestTemplateCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.TemplateCreate(t.Context(), c, client.Template{Host: "dup"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateGet ----

func TestTemplateGet_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcOK(t, []map[string]any{
			{
				"templateid":  "10",
				"host":        "Linux by Zabbix agent",
				"name":        "Linux by Zabbix agent",
				"description": "desc",
				"groups":      []map[string]any{{"groupid": "1"}},
				"macros": []map[string]any{
					{"macro": "{$AGENT.PORT}", "value": "10050"},
				},
				"parentTemplates": []map[string]any{
					{"templateid": "5"},
				},
			},
		}),
	})
	tmpl, err := client.TemplateGet(t.Context(), c, "10")
	if err != nil {
		t.Fatalf("TemplateGet: %v", err)
	}
	if tmpl == nil {
		t.Fatal("expected non-nil template")
	}
	if tmpl.TemplateID != "10" {
		t.Errorf("TemplateID = %q, want %q", tmpl.TemplateID, "10")
	}
	if tmpl.Host != "Linux by Zabbix agent" {
		t.Errorf("Host = %q, want %q", tmpl.Host, "Linux by Zabbix agent")
	}
	if tmpl.Description != "desc" {
		t.Errorf("Description = %q, want %q", tmpl.Description, "desc")
	}
	if len(tmpl.Groups) != 1 || tmpl.Groups[0].GroupID != "1" {
		t.Errorf("Groups = %v, want [{1}]", tmpl.Groups)
	}
	if len(tmpl.Macros) != 1 || tmpl.Macros[0].Macro != "{$AGENT.PORT}" || tmpl.Macros[0].Value != "10050" {
		t.Errorf("Macros = %v, want [{$AGENT.PORT} 10050]", tmpl.Macros)
	}
	if len(tmpl.ParentTemplates) != 1 || tmpl.ParentTemplates[0].TemplateID != "5" {
		t.Errorf("ParentTemplates = %v, want [{5}]", tmpl.ParentTemplates)
	}
}

func TestTemplateGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcOK(t, []map[string]any{}),
	})
	tmpl, err := client.TemplateGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl != nil {
		t.Errorf("expected nil for not-found, got %+v", tmpl)
	}
}

func TestTemplateGet_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.TemplateGet(t.Context(), c, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateGetByHost ----

func TestTemplateGetByHost_Single(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcOK(t, []map[string]any{
			{
				"templateid":      "10",
				"host":            "Linux by Zabbix agent",
				"name":            "Linux by Zabbix agent",
				"description":     "",
				"groups":          []map[string]any{},
				"macros":          []map[string]any{},
				"parentTemplates": []map[string]any{},
			},
		}),
	})
	templates, err := client.TemplateGetByHost(t.Context(), c, "Linux by Zabbix agent")
	if err != nil {
		t.Fatalf("TemplateGetByHost: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("len = %d, want 1", len(templates))
	}
	if templates[0].Host != "Linux by Zabbix agent" {
		t.Errorf("Host = %q, want %q", templates[0].Host, "Linux by Zabbix agent")
	}
}

func TestTemplateGetByHost_Empty(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcOK(t, []map[string]any{}),
	})
	templates, err := client.TemplateGetByHost(t.Context(), c, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected empty slice, got %v", templates)
	}
}

func TestTemplateGetByHost_Multiple(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcOK(t, []map[string]any{
			{"templateid": "1", "host": "Linux", "name": "Linux", "description": "", "groups": []map[string]any{}, "macros": []map[string]any{}, "parentTemplates": []map[string]any{}},
			{"templateid": "2", "host": "Linux", "name": "Linux", "description": "", "groups": []map[string]any{}, "macros": []map[string]any{}, "parentTemplates": []map[string]any{}},
		}),
	})
	templates, err := client.TemplateGetByHost(t.Context(), c, "Linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}
}

func TestTemplateGetByHost_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.get": rpcErr(t, -32500, "Application error."),
	})
	_, err := client.TemplateGetByHost(t.Context(), c, "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateUpdate ----

func TestTemplateUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.update": rpcOK(t, map[string]any{"templateids": []string{"10"}}),
	})
	if err := client.TemplateUpdate(t.Context(), c, client.Template{
		TemplateID:  "10",
		Host:        "Linux by Zabbix agent",
		Name:        "Linux by Zabbix agent",
		Description: "",
		Groups:      []client.TemplateGroupRef{{GroupID: "1"}},
		Macros:      []client.TemplateMacro{{Macro: "{$PORT}", Value: "161"}},
	}); err != nil {
		t.Fatalf("TemplateUpdate: %v", err)
	}
}

func TestTemplateUpdate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.update": rpcErr(t, -32602, "Invalid params."),
	})
	if err := client.TemplateUpdate(t.Context(), c, client.Template{TemplateID: "1"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- TemplateDelete ----

func TestTemplateDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.delete": rpcOK(t, map[string]any{"templateids": []string{"10"}}),
	})
	if err := client.TemplateDelete(t.Context(), c, "10"); err != nil {
		t.Fatalf("TemplateDelete: %v", err)
	}
}

func TestTemplateDelete_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"template.delete": rpcErr(t, -32500, "Cannot delete template."),
	})
	if err := client.TemplateDelete(t.Context(), c, "1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
