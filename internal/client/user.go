package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// User represents a Zabbix user.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
// gui_access, debug_mode, and users_status require getAccess:true in the request.
type User struct {
	UserID        string `json:"userid"`
	Username      string `json:"username"`
	Name          string `json:"name"`
	Surname       string `json:"surname"`
	URL           string `json:"url"`
	AutoLogin     string `json:"autologin"`
	AutoLogout    string `json:"autologout"`
	Language      string `json:"lang"`
	Refresh       string `json:"refresh"`
	Theme         string `json:"theme"`
	AttemptFailed string `json:"attempt_failed"`
	AttemptIP     string `json:"attempt_ip"`
	AttemptClock  string `json:"attempt_clock"`
	RowsPerPage   string `json:"rows_per_page"`
	Timezone      string `json:"timezone"`
	Provisioned   string `json:"provisioned"`
	GUIAccess     int64  `json:"gui_access,string"`
	DebugMode     int64  `json:"debug_mode,string"`
	UsersStatus   int64  `json:"users_status,string"`
	RoleID        string `json:"roleid"`
}

// UserGet fetches a user by ID. Returns nil if not found.
func UserGet(ctx context.Context, c Client, id string) (*User, error) {
	result, err := c.Call(ctx, "user.get", map[string]any{
		"userids":   []string{id},
		"output":    "extend",
		"getAccess": true,
		"limit":     1,
	})
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, fmt.Errorf("user.get: unexpected response: %w", err)
	}
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

// UserGetByUsername fetches users matching the given username.
func UserGetByUsername(ctx context.Context, c Client, username string) ([]User, error) {
	result, err := c.Call(ctx, "user.get", map[string]any{
		"filter":    map[string]any{"username": []string{username}},
		"output":    "extend",
		"getAccess": true,
	})
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, fmt.Errorf("user.get: unexpected response: %w", err)
	}
	return users, nil
}
