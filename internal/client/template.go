package client

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
)

// Template represents a Zabbix template.
type Template struct {
	TemplateID      string             `json:"templateid"`
	Host            string             `json:"host"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Groups          []TemplateGroupRef `json:"groups"`
	Macros          []TemplateMacro    `json:"macros"`
	ParentTemplates []TemplateRef      `json:"parentTemplates"`
}

// TemplateGroupRef is a reference to a template group used in template API calls.
type TemplateGroupRef struct {
	GroupID string `json:"groupid"`
}

// TemplateMacro is a user macro attached to a template.
type TemplateMacro struct {
	Macro string `json:"macro"`
	Value string `json:"value"`
}

// TemplateRef is a minimal template reference (ID only).
type TemplateRef struct {
	TemplateID string `json:"templateid"`
}

var templateSelectParams = map[string]any{
	"output":                "extend",
	"selectGroups":          "extend",
	"selectMacros":          "extend",
	"selectParentTemplates": "extend",
}

// TemplateCreate creates a new template and returns its ID.
func TemplateCreate(ctx context.Context, c Client, t Template) (string, error) {
	groups := make([]map[string]any, len(t.Groups))
	for i, g := range t.Groups {
		groups[i] = map[string]any{"groupid": g.GroupID}
	}
	params := map[string]any{
		"host":        t.Host,
		"name":        t.Name,
		"description": t.Description,
		"groups":      groups,
	}
	if len(t.Macros) > 0 {
		params["macros"] = t.Macros
	}
	if len(t.ParentTemplates) > 0 {
		templates := make([]map[string]any, len(t.ParentTemplates))
		for i, ref := range t.ParentTemplates {
			templates[i] = map[string]any{"templateid": ref.TemplateID}
		}
		params["templates"] = templates
	}
	result, err := c.Call(ctx, "template.create", params)
	if err != nil {
		return "", err
	}
	var out struct {
		TemplateIDs []string `json:"templateids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("template.create: unexpected response: %w", err)
	}
	if len(out.TemplateIDs) == 0 {
		return "", fmt.Errorf("template.create: empty templateids in response")
	}
	return out.TemplateIDs[0], nil
}

// TemplateGet fetches a template by ID. Returns nil if not found.
func TemplateGet(ctx context.Context, c Client, id string) (*Template, error) {
	params := map[string]any{"templateids": []string{id}, "limit": 1}
	maps.Copy(params, templateSelectParams)
	result, err := c.Call(ctx, "template.get", params)
	if err != nil {
		return nil, err
	}
	var templates []Template
	if err := json.Unmarshal(result, &templates); err != nil {
		return nil, fmt.Errorf("template.get: unexpected response: %w", err)
	}
	if len(templates) == 0 {
		return nil, nil
	}
	return &templates[0], nil
}

// TemplateGetByHost fetches templates matching the given technical name (host field).
func TemplateGetByHost(ctx context.Context, c Client, host string) ([]Template, error) {
	params := map[string]any{
		"filter": map[string]any{"host": []string{host}},
	}
	maps.Copy(params, templateSelectParams)
	result, err := c.Call(ctx, "template.get", params)
	if err != nil {
		return nil, err
	}
	var templates []Template
	if err := json.Unmarshal(result, &templates); err != nil {
		return nil, fmt.Errorf("template.get: unexpected response: %w", err)
	}
	return templates, nil
}

// TemplateUpdate updates an existing template.
func TemplateUpdate(ctx context.Context, c Client, t Template) error {
	groups := make([]map[string]any, len(t.Groups))
	for i, g := range t.Groups {
		groups[i] = map[string]any{"groupid": g.GroupID}
	}
	linkedTemplates := make([]map[string]any, len(t.ParentTemplates))
	for i, ref := range t.ParentTemplates {
		linkedTemplates[i] = map[string]any{"templateid": ref.TemplateID}
	}
	params := map[string]any{
		"templateid":  t.TemplateID,
		"host":        t.Host,
		"name":        t.Name,
		"description": t.Description,
		"groups":      groups,
		"macros":      t.Macros,
		"templates":   linkedTemplates,
	}
	_, err := c.Call(ctx, "template.update", params)
	return err
}

// TemplateDelete deletes a template by ID.
func TemplateDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "template.delete", []string{id})
	return err
}
