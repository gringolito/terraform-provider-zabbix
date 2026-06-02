package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// TemplateGroup represents a Zabbix template group.
type TemplateGroup struct {
	ID   string `json:"groupid"`
	Name string `json:"name"`
}

// TemplateGroupCreate creates a new template group and returns its ID.
func TemplateGroupCreate(ctx context.Context, c Client, name string) (string, error) {
	result, err := c.Call(ctx, "templategroup.create", map[string]any{"name": name})
	if err != nil {
		return "", err
	}
	var out struct {
		GroupIDs []string `json:"groupids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("templategroup.create: unexpected response: %w", err)
	}
	if len(out.GroupIDs) == 0 {
		return "", fmt.Errorf("templategroup.create: empty groupids in response")
	}
	return out.GroupIDs[0], nil
}

// TemplateGroupGet fetches a template group by ID. Returns nil if not found.
func TemplateGroupGet(ctx context.Context, c Client, id string) (*TemplateGroup, error) {
	result, err := c.Call(ctx, "templategroup.get", map[string]any{
		"groupids": []string{id},
		"output":   "extend",
		"limit":    1,
	})
	if err != nil {
		return nil, err
	}
	var groups []TemplateGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("templategroup.get: unexpected response: %w", err)
	}
	if len(groups) == 0 {
		return nil, nil
	}
	return &groups[0], nil
}

// TemplateGroupGetByName fetches template groups matching the given name.
func TemplateGroupGetByName(ctx context.Context, c Client, name string) ([]TemplateGroup, error) {
	result, err := c.Call(ctx, "templategroup.get", map[string]any{
		"filter": map[string]any{"name": []string{name}},
		"output": "extend",
	})
	if err != nil {
		return nil, err
	}
	var groups []TemplateGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("templategroup.get: unexpected response: %w", err)
	}
	return groups, nil
}

// TemplateGroupUpdate renames an existing template group.
func TemplateGroupUpdate(ctx context.Context, c Client, id, name string) error {
	_, err := c.Call(ctx, "templategroup.update", map[string]any{
		"groupid": id,
		"name":    name,
	})
	return err
}

// TemplateGroupDelete deletes a template group by ID.
func TemplateGroupDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "templategroup.delete", []string{id})
	return err
}
