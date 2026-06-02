package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// HostGroup represents a Zabbix host group.
type HostGroup struct {
	ID   string `json:"groupid"`
	Name string `json:"name"`
}

// HostGroupCreate creates a new host group and returns its ID.
func HostGroupCreate(ctx context.Context, c Client, name string) (string, error) {
	result, err := c.Call(ctx, "hostgroup.create", map[string]any{"name": name})
	if err != nil {
		return "", err
	}
	var out struct {
		GroupIDs []string `json:"groupids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("hostgroup.create: unexpected response: %w", err)
	}
	if len(out.GroupIDs) == 0 {
		return "", fmt.Errorf("hostgroup.create: empty groupids in response")
	}
	return out.GroupIDs[0], nil
}

// HostGroupGet fetches a host group by ID. Returns nil if not found.
func HostGroupGet(ctx context.Context, c Client, id string) (*HostGroup, error) {
	result, err := c.Call(ctx, "hostgroup.get", map[string]any{
		"groupids": []string{id},
		"output":   "extend",
		"limit":    1,
	})
	if err != nil {
		return nil, err
	}
	var groups []HostGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("hostgroup.get: unexpected response: %w", err)
	}
	if len(groups) == 0 {
		return nil, nil
	}
	return &groups[0], nil
}

// HostGroupGetByName fetches host groups matching the given name.
func HostGroupGetByName(ctx context.Context, c Client, name string) ([]HostGroup, error) {
	result, err := c.Call(ctx, "hostgroup.get", map[string]any{
		"filter": map[string]any{"name": []string{name}},
		"output": "extend",
	})
	if err != nil {
		return nil, err
	}
	var groups []HostGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("hostgroup.get: unexpected response: %w", err)
	}
	return groups, nil
}

// HostGroupUpdate renames an existing host group.
func HostGroupUpdate(ctx context.Context, c Client, id, name string) error {
	_, err := c.Call(ctx, "hostgroup.update", map[string]any{
		"groupid": id,
		"name":    name,
	})
	return err
}

// HostGroupDelete deletes a host group by ID.
func HostGroupDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "hostgroup.delete", []string{id})
	return err
}
