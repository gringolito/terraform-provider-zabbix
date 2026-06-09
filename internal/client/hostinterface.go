package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// HostInterfaceSNMPDetails holds SNMP-specific settings for a host interface.
// Zabbix 7.0 returns integer fields as JSON strings.
type HostInterfaceSNMPDetails struct {
	Version        int64  `json:"version,string"`
	Community      string `json:"community"`
	BulkRequests   int64  `json:"bulk,string"`
	SecurityName   string `json:"securityname"`
	SecurityLevel  int64  `json:"securitylevel,string"`
	AuthProtocol   int64  `json:"authprotocol,string"`
	AuthPassphrase string `json:"authpassphrase"`
	PrivProtocol   int64  `json:"privprotocol,string"`
	PrivPassphrase string `json:"privpassphrase"`
	ContextName    string `json:"contextname"`
}

// HostInterface represents a Zabbix host interface.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
// The Details field is nil for non-SNMP interfaces (Zabbix returns [] in that case).
type HostInterface struct {
	InterfaceID string                    `json:"interfaceid"`
	HostID      string                    `json:"hostid"`
	Type        int64                     `json:"type,string"`
	UseIP       int64                     `json:"useip,string"`
	IP          string                    `json:"ip"`
	DNS         string                    `json:"dns"`
	Port        string                    `json:"port"`
	Main        int64                     `json:"main,string"`
	Details     *HostInterfaceSNMPDetails `json:"-"`
}

// UnmarshalJSON handles Zabbix's mixed details encoding: [] for non-SNMP, object for SNMP.
func (hi *HostInterface) UnmarshalJSON(data []byte) error {
	type alias struct {
		InterfaceID string          `json:"interfaceid"`
		HostID      string          `json:"hostid"`
		Type        int64           `json:"type,string"`
		UseIP       int64           `json:"useip,string"`
		IP          string          `json:"ip"`
		DNS         string          `json:"dns"`
		Port        string          `json:"port"`
		Main        int64           `json:"main,string"`
		Details     json.RawMessage `json:"details"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	hi.InterfaceID = a.InterfaceID
	hi.HostID = a.HostID
	hi.Type = a.Type
	hi.UseIP = a.UseIP
	hi.IP = a.IP
	hi.DNS = a.DNS
	hi.Port = a.Port
	hi.Main = a.Main

	if len(a.Details) > 0 && string(a.Details) != "[]" && string(a.Details) != "{}" {
		hi.Details = &HostInterfaceSNMPDetails{}
		if err := json.Unmarshal(a.Details, hi.Details); err != nil {
			return fmt.Errorf("hostinterface details: %w", err)
		}
	}
	return nil
}

func hostInterfaceParams(hi HostInterface) map[string]any {
	params := map[string]any{
		"hostid": hi.HostID,
		"type":   hi.Type,
		"useip":  hi.UseIP,
		"ip":     hi.IP,
		"dns":    hi.DNS,
		"port":   hi.Port,
		"main":   hi.Main,
	}
	if hi.Details != nil {
		params["details"] = map[string]any{
			"version":        hi.Details.Version,
			"community":      hi.Details.Community,
			"bulk":           hi.Details.BulkRequests,
			"securityname":   hi.Details.SecurityName,
			"securitylevel":  hi.Details.SecurityLevel,
			"authprotocol":   hi.Details.AuthProtocol,
			"authpassphrase": hi.Details.AuthPassphrase,
			"privprotocol":   hi.Details.PrivProtocol,
			"privpassphrase": hi.Details.PrivPassphrase,
			"contextname":    hi.Details.ContextName,
		}
	}
	return params
}

// HostInterfaceCreate creates a new host interface and returns its ID.
func HostInterfaceCreate(ctx context.Context, c Client, hi HostInterface) (string, error) {
	result, err := c.Call(ctx, "hostinterface.create", hostInterfaceParams(hi))
	if err != nil {
		return "", err
	}
	var out struct {
		InterfaceIDs []string `json:"interfaceids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("hostinterface.create: unexpected response: %w", err)
	}
	if len(out.InterfaceIDs) == 0 {
		return "", fmt.Errorf("hostinterface.create: empty interfaceids in response")
	}
	return out.InterfaceIDs[0], nil
}

// HostInterfaceGet fetches a host interface by ID. Returns nil if not found.
func HostInterfaceGet(ctx context.Context, c Client, id string) (*HostInterface, error) {
	params := map[string]any{
		"interfaceids": []string{id},
		"output":       "extend",
		"limit":        1,
	}
	result, err := c.Call(ctx, "hostinterface.get", params)
	if err != nil {
		return nil, err
	}
	var ifaces []HostInterface
	if err := json.Unmarshal(result, &ifaces); err != nil {
		return nil, fmt.Errorf("hostinterface.get: unexpected response: %w", err)
	}
	if len(ifaces) == 0 {
		return nil, nil
	}
	return &ifaces[0], nil
}

// HostInterfaceGetByHostAndType fetches all interfaces of the given type for the given host.
func HostInterfaceGetByHostAndType(ctx context.Context, c Client, hostID string, ifaceType int64) ([]HostInterface, error) {
	params := map[string]any{
		"hostids": []string{hostID},
		"filter":  map[string]any{"type": []int64{ifaceType}},
		"output":  "extend",
	}
	result, err := c.Call(ctx, "hostinterface.get", params)
	if err != nil {
		return nil, err
	}
	var ifaces []HostInterface
	if err := json.Unmarshal(result, &ifaces); err != nil {
		return nil, fmt.Errorf("hostinterface.get: unexpected response: %w", err)
	}
	return ifaces, nil
}

// HostInterfaceUpdate updates an existing host interface.
func HostInterfaceUpdate(ctx context.Context, c Client, hi HostInterface) error {
	params := hostInterfaceParams(hi)
	params["interfaceid"] = hi.InterfaceID
	_, err := c.Call(ctx, "hostinterface.update", params)
	return err
}

// HostInterfaceDelete deletes a host interface by ID.
func HostInterfaceDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "hostinterface.delete", []string{id})
	return err
}
