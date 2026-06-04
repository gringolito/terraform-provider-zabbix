package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// UserGroup represents a Zabbix user group.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type UserGroup struct {
	ID          string `json:"usrgrpid"`
	Name        string `json:"name"`
	GUIAccess   int64  `json:"gui_access,string"`
	DebugMode   int64  `json:"debug_mode,string"`
	UsersStatus int64  `json:"users_status,string"`
}

// UserGroupCreate creates a new user group and returns its ID.
func UserGroupCreate(ctx context.Context, c Client, ug UserGroup) (string, error) {
	result, err := c.Call(ctx, "usergroup.create", map[string]any{
		"name":         ug.Name,
		"gui_access":   ug.GUIAccess,
		"debug_mode":   ug.DebugMode,
		"users_status": ug.UsersStatus,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		GroupIDs []string `json:"usrgrpids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("usergroup.create: unexpected response: %w", err)
	}
	if len(out.GroupIDs) == 0 {
		return "", fmt.Errorf("usergroup.create: empty usrgrpids in response")
	}
	return out.GroupIDs[0], nil
}

// UserGroupGet fetches a user group by ID. Returns nil if not found.
func UserGroupGet(ctx context.Context, c Client, id string) (*UserGroup, error) {
	result, err := c.Call(ctx, "usergroup.get", map[string]any{
		"usrgrpids": []string{id},
		"output":    "extend",
		"limit":     1,
	})
	if err != nil {
		return nil, err
	}
	var groups []UserGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("usergroup.get: unexpected response: %w", err)
	}
	if len(groups) == 0 {
		return nil, nil
	}
	return &groups[0], nil
}

// UserGroupGetByName fetches user groups matching the given name.
func UserGroupGetByName(ctx context.Context, c Client, name string) ([]UserGroup, error) {
	result, err := c.Call(ctx, "usergroup.get", map[string]any{
		"filter": map[string]any{"name": []string{name}},
		"output": "extend",
	})
	if err != nil {
		return nil, err
	}
	var groups []UserGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, fmt.Errorf("usergroup.get: unexpected response: %w", err)
	}
	return groups, nil
}

// UserGroupUpdate updates an existing user group.
func UserGroupUpdate(ctx context.Context, c Client, ug UserGroup) error {
	_, err := c.Call(ctx, "usergroup.update", map[string]any{
		"usrgrpid":     ug.ID,
		"name":         ug.Name,
		"gui_access":   ug.GUIAccess,
		"debug_mode":   ug.DebugMode,
		"users_status": ug.UsersStatus,
	})
	return err
}

// UserGroupDelete deletes a user group by ID.
func UserGroupDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "usergroup.delete", []string{id})
	return err
}
