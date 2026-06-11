package client

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
)

// HostGroupRef is a minimal host group reference used in host create/update/read.
type HostGroupRef struct {
	GroupID string `json:"groupid"`
}

// HostTag is a key/value tag attached to a host.
type HostTag struct {
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// Inventory holds host inventory key/value pairs.
// Zabbix returns [] (empty JSON array) when inventory mode is disabled,
// and an object otherwise; the custom unmarshaler normalises both to a map.
type Inventory map[string]string

func (inv *Inventory) UnmarshalJSON(data []byte) error {
	if string(data) == "[]" {
		*inv = nil
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*inv = m
	return nil
}

// Host represents a Zabbix host.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type Host struct {
	HostID        string         `json:"hostid"`
	Host          string         `json:"host"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Status        int64          `json:"status,string"`
	Groups        []HostGroupRef `json:"groups"`
	Tags          []HostTag      `json:"tags"`
	Inventory     Inventory      `json:"inventory"`
	InventoryMode int64          `json:"inventory_mode,string"`
	ProxyID       string         `json:"proxyid"`
}

var hostSelectParams = map[string]any{
	"output":          "extend",
	"selectGroups":    "extend",
	"selectTags":      "extend",
	"selectInventory": "extend",
}

// HostCreate creates a new host and returns its ID.
func HostCreate(ctx context.Context, c Client, h Host) (string, error) {
	groups := make([]map[string]any, len(h.Groups))
	for i, g := range h.Groups {
		groups[i] = map[string]any{"groupid": g.GroupID}
	}
	params := map[string]any{
		"host":           h.Host,
		"groups":         groups,
		"name":           h.Name,
		"description":    h.Description,
		"status":         h.Status,
		"inventory_mode": h.InventoryMode,
		"proxyid":        h.ProxyID,
	}
	if len(h.Tags) > 0 {
		params["tags"] = h.Tags
	}
	if len(h.Inventory) > 0 {
		params["inventory"] = h.Inventory
	}
	result, err := c.Call(ctx, "host.create", params)
	if err != nil {
		return "", err
	}
	var out struct {
		HostIDs []string `json:"hostids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("host.create: unexpected response: %w", err)
	}
	if len(out.HostIDs) == 0 {
		return "", fmt.Errorf("host.create: empty hostids in response")
	}
	return out.HostIDs[0], nil
}

// HostGet fetches a host by ID. Returns nil if not found.
func HostGet(ctx context.Context, c Client, id string) (*Host, error) {
	params := map[string]any{"hostids": []string{id}, "limit": 1}
	maps.Copy(params, hostSelectParams)
	result, err := c.Call(ctx, "host.get", params)
	if err != nil {
		return nil, err
	}
	var hosts []Host
	if err := json.Unmarshal(result, &hosts); err != nil {
		return nil, fmt.Errorf("host.get: unexpected response: %w", err)
	}
	if len(hosts) == 0 {
		return nil, nil
	}
	return &hosts[0], nil
}

// HostGetByTechnicalName fetches hosts matching the given technical name.
func HostGetByTechnicalName(ctx context.Context, c Client, name string) ([]Host, error) {
	params := map[string]any{
		"filter": map[string]any{"host": []string{name}},
	}
	maps.Copy(params, hostSelectParams)
	result, err := c.Call(ctx, "host.get", params)
	if err != nil {
		return nil, err
	}
	var hosts []Host
	if err := json.Unmarshal(result, &hosts); err != nil {
		return nil, fmt.Errorf("host.get: unexpected response: %w", err)
	}
	return hosts, nil
}

// HostUpdate updates an existing host.
func HostUpdate(ctx context.Context, c Client, h Host) error {
	groups := make([]map[string]any, len(h.Groups))
	for i, g := range h.Groups {
		groups[i] = map[string]any{"groupid": g.GroupID}
	}
	params := map[string]any{
		"hostid":         h.HostID,
		"host":           h.Host,
		"groups":         groups,
		"name":           h.Name,
		"description":    h.Description,
		"status":         h.Status,
		"inventory_mode": h.InventoryMode,
		"proxyid":        h.ProxyID,
		"tags":           h.Tags,
	}
	if len(h.Inventory) > 0 {
		params["inventory"] = h.Inventory
	}
	_, err := c.Call(ctx, "host.update", params)
	return err
}

// HostDelete deletes a host by ID.
func HostDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "host.delete", []string{id})
	return err
}

// HostTemplateLinkAdd links templates to a host via host.massadd.
func HostTemplateLinkAdd(ctx context.Context, c Client, hostID string, templateIDs []string) error {
	templates := make([]map[string]any, len(templateIDs))
	for i, id := range templateIDs {
		templates[i] = map[string]any{"templateid": id}
	}
	params := map[string]any{
		"hosts":     []map[string]any{{"hostid": hostID}},
		"templates": templates,
	}
	_, err := c.Call(ctx, "host.massadd", params)
	return err
}

// HostTemplateLinkRemove unlinks a template from a host via host.massremove.
// doClear=true sends templateids_clear (deletes inherited items); false sends templateids_link.
func HostTemplateLinkRemove(ctx context.Context, c Client, hostID, templateID string, doClear bool) error {
	key := "templateids"
	if doClear {
		key = "templateids_clear"
	}
	params := map[string]any{
		"hostids": []string{hostID},
		key:       []string{templateID},
	}
	_, err := c.Call(ctx, "host.massremove", params)
	return err
}

// HostGetTemplates returns the templates currently linked to a host.
// Returns nil, nil if the host does not exist.
func HostGetTemplates(ctx context.Context, c Client, hostID string) ([]TemplateRef, error) {
	params := map[string]any{
		"hostids":               []string{hostID},
		"output":                "hostid",
		"selectParentTemplates": "extend",
		"limit":                 1,
	}
	result, err := c.Call(ctx, "host.get", params)
	if err != nil {
		return nil, err
	}
	var hosts []struct {
		ParentTemplates []TemplateRef `json:"parentTemplates"`
	}
	if err := json.Unmarshal(result, &hosts); err != nil {
		return nil, fmt.Errorf("host.get: unexpected response: %w", err)
	}
	if len(hosts) == 0 {
		return nil, nil
	}
	return hosts[0].ParentTemplates, nil
}
