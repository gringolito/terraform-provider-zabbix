package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Item represents a Zabbix item (read-only; full CRUD is out of scope).
type Item struct {
	ItemID string `json:"itemid"`
	Key    string `json:"key_"`
	Name   string `json:"name"`
	HostID string `json:"hostid"`
}

// ItemGet fetches an item by ID. Returns nil if not found.
func ItemGet(ctx context.Context, c Client, id string) (*Item, error) {
	params := map[string]any{
		"itemids": []string{id},
		"output":  []string{"itemid", "key_", "name", "hostid"},
		"limit":   1,
	}
	result, err := c.Call(ctx, "item.get", params)
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("item.get: unexpected response: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// ItemGetByKeyAndScope fetches items by key_ scoped to a host or template.
// Exactly one of hostID or templateID must be non-empty.
func ItemGetByKeyAndScope(ctx context.Context, c Client, key, hostID, templateID string) ([]Item, error) {
	params := map[string]any{
		"filter": map[string]any{"key_": []string{key}},
		"output": []string{"itemid", "key_", "name", "hostid"},
	}
	if hostID != "" {
		params["hostids"] = []string{hostID}
	} else if templateID != "" {
		params["templateids"] = []string{templateID}
	}
	result, err := c.Call(ctx, "item.get", params)
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("item.get: unexpected response: %w", err)
	}
	return items, nil
}
