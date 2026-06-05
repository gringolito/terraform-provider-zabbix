package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// RuleUI represents a UI element rule within a Zabbix role.
type RuleUI struct {
	Name   string `json:"name"`
	Status int64  `json:"status,string"`
}

// RuleModule represents a module rule within a Zabbix role.
// Zabbix returns moduleid as a JSON integer (not a string).
type RuleModule struct {
	ModuleID int64 `json:"moduleid"`
	Status   int64 `json:"status,string"`
}

// RuleAction represents an action rule within a Zabbix role.
type RuleAction struct {
	Name   string `json:"name"`
	Status int64  `json:"status,string"`
}

// RoleRules holds the access rules for a Zabbix role.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings, except moduleid.
type RoleRules struct {
	UI                   []RuleUI     `json:"ui"`
	UIDefaultAccess      int64        `json:"ui.default_access,string"`
	ModulesDefaultAccess int64        `json:"modules.default_access,string"`
	ActionsDefaultAccess int64        `json:"actions.default_access,string"`
	APIAccess            int64        `json:"api.access,string"`
	APIMode              int64        `json:"api.mode,string"`
	APIMethods           []string     `json:"api"`
	Modules              []RuleModule `json:"modules"`
	Actions              []RuleAction `json:"actions"`
}

// Role represents a Zabbix role.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type Role struct {
	ID       string    `json:"roleid"`
	Name     string    `json:"name"`
	Type     int64     `json:"type,string"`
	ReadOnly int64     `json:"readonly,string"`
	Rules    RoleRules `json:"rules"`
	// HasRules controls whether Rules is included in create/update params.
	HasRules bool `json:"-"`
}

// RoleCreate creates a new role and returns its ID.
func RoleCreate(ctx context.Context, c Client, r Role) (string, error) {
	params := map[string]any{
		"name": r.Name,
		"type": r.Type,
	}
	if r.HasRules {
		params["rules"] = roleRulesToParams(r.Rules)
	}
	result, err := c.Call(ctx, "role.create", params)
	if err != nil {
		return "", err
	}
	var out struct {
		RoleIDs []string `json:"roleids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("role.create: unexpected response: %w", err)
	}
	if len(out.RoleIDs) == 0 {
		return "", fmt.Errorf("role.create: empty roleids in response")
	}
	return out.RoleIDs[0], nil
}

// RoleGet fetches a role by ID (with rules). Returns nil if not found.
func RoleGet(ctx context.Context, c Client, id string) (*Role, error) {
	result, err := c.Call(ctx, "role.get", map[string]any{
		"roleids":     []string{id},
		"output":      "extend",
		"selectRules": "extend",
		"limit":       1,
	})
	if err != nil {
		return nil, err
	}
	var roles []Role
	if err := json.Unmarshal(result, &roles); err != nil {
		return nil, fmt.Errorf("role.get: unexpected response: %w", err)
	}
	if len(roles) == 0 {
		return nil, nil
	}
	return &roles[0], nil
}

// RoleGetByName fetches roles matching the given name (with rules).
func RoleGetByName(ctx context.Context, c Client, name string) ([]Role, error) {
	result, err := c.Call(ctx, "role.get", map[string]any{
		"filter":      map[string]any{"name": []string{name}},
		"output":      "extend",
		"selectRules": "extend",
	})
	if err != nil {
		return nil, err
	}
	var roles []Role
	if err := json.Unmarshal(result, &roles); err != nil {
		return nil, fmt.Errorf("role.get: unexpected response: %w", err)
	}
	return roles, nil
}

// RoleUpdate updates an existing role.
func RoleUpdate(ctx context.Context, c Client, r Role) error {
	params := map[string]any{
		"roleid": r.ID,
		"name":   r.Name,
		"type":   r.Type,
	}
	if r.HasRules {
		params["rules"] = roleRulesToParams(r.Rules)
	}
	_, err := c.Call(ctx, "role.update", params)
	return err
}

// RoleDelete deletes a role by ID.
func RoleDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "role.delete", []string{id})
	return err
}

func roleRulesToParams(rules RoleRules) map[string]any {
	ui := make([]map[string]any, len(rules.UI))
	for i, u := range rules.UI {
		ui[i] = map[string]any{"name": u.Name, "status": u.Status}
	}
	modules := make([]map[string]any, len(rules.Modules))
	for i, m := range rules.Modules {
		modules[i] = map[string]any{"moduleid": m.ModuleID, "status": m.Status}
	}
	actions := make([]map[string]any, len(rules.Actions))
	for i, a := range rules.Actions {
		actions[i] = map[string]any{"name": a.Name, "status": a.Status}
	}
	methods := rules.APIMethods
	if methods == nil {
		methods = []string{}
	}
	return map[string]any{
		"ui":                     ui,
		"ui.default_access":      rules.UIDefaultAccess,
		"modules.default_access": rules.ModulesDefaultAccess,
		"api.access":             rules.APIAccess,
		"api.mode":               rules.APIMode,
		"api":                    methods,
		"modules":                modules,
		"actions":                actions,
		"actions.default_access": rules.ActionsDefaultAccess,
	}
}
