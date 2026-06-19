package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// User represents a Zabbix user.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type User struct {
	UserID   string `json:"userid"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Type     int64  `json:"type,string"`
	RoleID   string `json:"roleid"`
}

// UserGet fetches a user by ID. Returns nil if not found.
func UserGet(ctx context.Context, c Client, id string) (*User, error) {
	result, err := c.Call(ctx, "user.get", map[string]any{
		"userids": []string{id},
		"output":  "extend",
		"limit":   1,
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
		"filter": map[string]any{"username": []string{username}},
		"output": "extend",
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
