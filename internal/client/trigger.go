package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// TriggerTag represents a single Zabbix trigger tag.
type TriggerTag struct {
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// Trigger represents a Zabbix trigger.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type Trigger struct {
	TriggerID          string       `json:"triggerid"`
	Description        string       `json:"description"`
	Expression         string       `json:"expression"`
	RecoveryMode       int64        `json:"recovery_mode,string"`
	RecoveryExpression string       `json:"recovery_expression"`
	Priority           int64        `json:"priority,string"`
	Status             int64        `json:"status,string"`
	ManualClose        int64        `json:"manual_close,string"`
	Comments           string       `json:"comments"`
	URL                string       `json:"url"`
	Tags               []TriggerTag `json:"tags"`
}

func triggerParams(tr Trigger) map[string]any {
	params := map[string]any{
		"description":         tr.Description,
		"expression":          tr.Expression,
		"recovery_mode":       tr.RecoveryMode,
		"recovery_expression": tr.RecoveryExpression,
		"priority":            tr.Priority,
		"status":              tr.Status,
		"manual_close":        tr.ManualClose,
		"comments":            tr.Comments,
		"url":                 tr.URL,
		"tags":                tr.Tags,
	}
	return params
}

// TriggerCreate creates a new trigger and returns its ID.
func TriggerCreate(ctx context.Context, c Client, tr Trigger) (string, error) {
	result, err := c.Call(ctx, "trigger.create", triggerParams(tr))
	if err != nil {
		return "", err
	}
	var out struct {
		TriggerIDs []string `json:"triggerids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("trigger.create: unexpected response: %w", err)
	}
	if len(out.TriggerIDs) == 0 {
		return "", fmt.Errorf("trigger.create: empty triggerids in response")
	}
	return out.TriggerIDs[0], nil
}

// TriggerGet fetches a trigger by ID. Returns nil if not found.
func TriggerGet(ctx context.Context, c Client, id string) (*Trigger, error) {
	params := map[string]any{
		"triggerids":       []string{id},
		"output":           "extend",
		"selectTags":       "extend",
		"expandExpression": true,
		"limit":            1,
	}
	result, err := c.Call(ctx, "trigger.get", params)
	if err != nil {
		return nil, err
	}
	var triggers []Trigger
	if err := json.Unmarshal(result, &triggers); err != nil {
		return nil, fmt.Errorf("trigger.get: unexpected response: %w", err)
	}
	if len(triggers) == 0 {
		return nil, nil
	}
	return &triggers[0], nil
}

// TriggerGetByDescriptionAndScope fetches triggers by description scoped to a host or template.
// Exactly one of hostID or templateID must be non-empty.
func TriggerGetByDescriptionAndScope(ctx context.Context, c Client, description, hostID, templateID string) ([]Trigger, error) {
	params := map[string]any{
		"filter":           map[string]any{"description": []string{description}},
		"output":           "extend",
		"selectTags":       "extend",
		"expandExpression": true,
	}
	if hostID != "" {
		params["hostids"] = []string{hostID}
	} else if templateID != "" {
		params["templateids"] = []string{templateID}
	}
	result, err := c.Call(ctx, "trigger.get", params)
	if err != nil {
		return nil, err
	}
	var triggers []Trigger
	if err := json.Unmarshal(result, &triggers); err != nil {
		return nil, fmt.Errorf("trigger.get: unexpected response: %w", err)
	}
	return triggers, nil
}

// TriggerUpdate updates an existing trigger.
func TriggerUpdate(ctx context.Context, c Client, tr Trigger) error {
	params := triggerParams(tr)
	params["triggerid"] = tr.TriggerID
	_, err := c.Call(ctx, "trigger.update", params)
	return err
}

// TriggerDelete deletes a trigger by ID.
func TriggerDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "trigger.delete", []string{id})
	return err
}
