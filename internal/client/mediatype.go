package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type MediaTypeType int

const (
	MediaTypeTypeEmail   MediaTypeType = 0
	MediaTypeTypeScript  MediaTypeType = 1
	MediaTypeTypeSMS     MediaTypeType = 2
	MediaTypeTypeWebhook MediaTypeType = 4
)

type MediaTypeStatus int

const (
	MediaTypeStatusEnabled  MediaTypeStatus = 0
	MediaTypeStatusDisabled MediaTypeStatus = 1
)

// MediaTypeParameter is a key/value pair passed to a webhook media type.
type MediaTypeParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MessageTemplate is a per-event-source notification template.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type MessageTemplate struct {
	EventSource int    `json:"eventsource,string"`
	Recovery    int    `json:"recovery,string"`
	Subject     string `json:"subject"`
	Message     string `json:"message"`
}

// MediaType represents a Zabbix media type.
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type MediaType struct {
	ID              string          `json:"mediatypeid"`
	Name            string          `json:"name"`
	Type            MediaTypeType   `json:"type,string"`
	Status          MediaTypeStatus `json:"status,string"`
	Description     string          `json:"description"`
	MaxSessions     int             `json:"maxsessions,string"`
	MaxAttempts     int             `json:"maxattempts,string"`
	AttemptInterval string          `json:"attempt_interval"`
	// Email settings
	SMTPServer         string `json:"smtp_server"`
	SMTPPort           int    `json:"smtp_port,string"`
	SMTPHelo           string `json:"smtp_helo"`
	SMTPEmail          string `json:"smtp_email"`
	SMTPSecurity       int    `json:"smtp_security,string"`
	SMTPAuthentication int    `json:"smtp_authentication,string"`
	Username           string `json:"username"`
	Passwd             string `json:"passwd"`
	ContentType        int    `json:"content_type,string"`
	// SMS settings
	GSMModem string `json:"gsm_modem"`
	// Script settings
	ExecPath   string `json:"exec_path"`
	ExecParams string `json:"exec_params"`
	// Webhook settings
	Script        string               `json:"script"`
	Timeout       string               `json:"timeout"`
	ProcessTags   int                  `json:"process_tags,string"`
	ShowEventMenu int                  `json:"show_event_menu,string"`
	EventMenuURL  string               `json:"event_menu_url"`
	EventMenuName string               `json:"event_menu_name"`
	Parameters    []MediaTypeParameter `json:"parameters"`
	// Common
	MessageTemplates []MessageTemplate `json:"message_templates"`
}

// mediaTypeParams builds the API params map for create/update, selecting only
// the fields relevant to the media type (avoids sending stale zeros for other types).
func mediaTypeParams(mt MediaType) map[string]any {
	params := map[string]any{
		"name":             mt.Name,
		"type":             mt.Type,
		"status":           mt.Status,
		"description":      mt.Description,
		"maxsessions":      mt.MaxSessions,
		"maxattempts":      mt.MaxAttempts,
		"attempt_interval": mt.AttemptInterval,
	}
	if mt.MessageTemplates != nil {
		// Build message_templates as []map[string]any so integers are marshaled
		// as proper JSON numbers, not strings (the struct uses ,string for reads).
		tmpl := make([]map[string]any, len(mt.MessageTemplates))
		for i, t := range mt.MessageTemplates {
			tmpl[i] = map[string]any{
				"eventsource": t.EventSource,
				"recovery":    t.Recovery,
				"subject":     t.Subject,
				"message":     t.Message,
			}
		}
		params["message_templates"] = tmpl
	}
	switch mt.Type {
	case MediaTypeTypeEmail:
		params["smtp_server"] = mt.SMTPServer
		params["smtp_port"] = mt.SMTPPort
		params["smtp_helo"] = mt.SMTPHelo
		params["smtp_email"] = mt.SMTPEmail
		params["smtp_security"] = mt.SMTPSecurity
		params["smtp_authentication"] = mt.SMTPAuthentication
		params["username"] = mt.Username
		params["passwd"] = mt.Passwd
		params["content_type"] = mt.ContentType
	case MediaTypeTypeSMS:
		params["gsm_modem"] = mt.GSMModem
	case MediaTypeTypeScript:
		params["exec_path"] = mt.ExecPath
		params["exec_params"] = mt.ExecParams
	case MediaTypeTypeWebhook:
		params["script"] = mt.Script
		params["timeout"] = mt.Timeout
		params["process_tags"] = mt.ProcessTags
		params["show_event_menu"] = mt.ShowEventMenu
		params["event_menu_url"] = mt.EventMenuURL
		params["event_menu_name"] = mt.EventMenuName
		if mt.Parameters != nil {
			params["parameters"] = mt.Parameters
		}
	}
	return params
}

// MediaTypeCreate creates a new media type and returns its ID.
func MediaTypeCreate(ctx context.Context, c Client, mt MediaType) (string, error) {
	result, err := c.Call(ctx, "mediatype.create", mediaTypeParams(mt))
	if err != nil {
		return "", err
	}
	var out struct {
		MediaTypeIDs []string `json:"mediatypeids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("mediatype.create: unexpected response: %w", err)
	}
	if len(out.MediaTypeIDs) == 0 {
		return "", fmt.Errorf("mediatype.create: empty mediatypeids in response")
	}
	return out.MediaTypeIDs[0], nil
}

// MediaTypeGet fetches a media type by ID. Returns nil if not found.
func MediaTypeGet(ctx context.Context, c Client, id string) (*MediaType, error) {
	result, err := c.Call(ctx, "mediatype.get", map[string]any{
		"mediatypeids":           []string{id},
		"output":                 "extend",
		"selectMessageTemplates": "extend",
		"limit":                  1,
	})
	if err != nil {
		return nil, err
	}
	var mts []MediaType
	if err := json.Unmarshal(result, &mts); err != nil {
		return nil, fmt.Errorf("mediatype.get: unexpected response: %w", err)
	}
	if len(mts) == 0 {
		return nil, nil
	}
	return &mts[0], nil
}

// MediaTypeGetByName fetches media types matching the given name.
func MediaTypeGetByName(ctx context.Context, c Client, name string) ([]MediaType, error) {
	result, err := c.Call(ctx, "mediatype.get", map[string]any{
		"filter":                 map[string]any{"name": []string{name}},
		"output":                 "extend",
		"selectMessageTemplates": "extend",
	})
	if err != nil {
		return nil, err
	}
	var mts []MediaType
	if err := json.Unmarshal(result, &mts); err != nil {
		return nil, fmt.Errorf("mediatype.get: unexpected response: %w", err)
	}
	return mts, nil
}

// MediaTypeUpdate updates an existing media type.
func MediaTypeUpdate(ctx context.Context, c Client, mt MediaType) error {
	params := mediaTypeParams(mt)
	params["mediatypeid"] = mt.ID
	_, err := c.Call(ctx, "mediatype.update", params)
	return err
}

// MediaTypeDelete deletes a media type by ID.
func MediaTypeDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "mediatype.delete", []string{id})
	return err
}
