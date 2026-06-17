package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Action represents a Zabbix trigger action (eventsource = 0).
// Zabbix 7.0 JSON-RPC returns integer fields as JSON strings.
type Action struct {
	ActionID           string            `json:"actionid,omitempty"`
	Name               string            `json:"name"`
	Status             int64             `json:"status,string"`
	EscPeriod          string            `json:"esc_period"`
	PauseSuppressed    int64             `json:"pause_suppressed,string"`
	NotifyIfCanceled   int64             `json:"notify_if_canceled,string"`
	DefShortdata       string            `json:"def_shortdata"`
	DefLongdata        string            `json:"def_longdata"`
	Filter             ActionFilter      `json:"filter"`
	Operations         []ActionOperation `json:"operations"`
	RecoveryOperations []ActionOperation `json:"recovery_operations"`
	UpdateOperations   []ActionOperation `json:"update_operations"`
}

// ActionFilter holds the filter block of an Action.
type ActionFilter struct {
	EvalType   int64             `json:"evaltype,string"`
	Formula    string            `json:"formula"`
	Conditions []ActionCondition `json:"conditions"`
}

// ActionCondition is a single condition in an ActionFilter.
type ActionCondition struct {
	ConditionType int64  `json:"conditiontype,string"`
	Operator      int64  `json:"operator,string"`
	Value         string `json:"value"`
	Value2        string `json:"value2"`
	FormulaID     string `json:"formulaid"`
}

// ActionOperation is a step in the operation, recovery_operation, or
// update_operation list of an Action.
type ActionOperation struct {
	OperationType int64                    `json:"operationtype,string"`
	EscPeriod     string                   `json:"esc_period,omitempty"`
	EscStepFrom   int64                    `json:"esc_step_from,string,omitempty"`
	EscStepTo     int64                    `json:"esc_step_to,string,omitempty"`
	OpMessage     *ActionOpMessage         `json:"opmessage,omitempty"`
	OpMessageGrp  []ActionOpRecipientGroup `json:"opmessage_grp,omitempty"`
	OpMessageUsr  []ActionOpRecipientUser  `json:"opmessage_usr,omitempty"`
	OpCommand     *ActionOpCommand         `json:"opcommand,omitempty"`
	OpCommandHst  []ActionOpCommandHost    `json:"opcommand_hst,omitempty"`
	OpCommandGrp  []ActionOpCommandGroup   `json:"opcommand_grp,omitempty"`
}

// ActionOpMessage holds the send-message parameters for an ActionOperation.
type ActionOpMessage struct {
	UseDefault  int64  `json:"default_msg,string"`
	Subject     string `json:"subject"`
	Message     string `json:"message"`
	MediaTypeID string `json:"mediatypeid"`
}

// ActionOpRecipientGroup is a user group recipient on a send-message operation.
type ActionOpRecipientGroup struct {
	UserGroupID string `json:"usrgrpid"`
}

// ActionOpRecipientUser is a user recipient on a send-message operation.
type ActionOpRecipientUser struct {
	UserID string `json:"userid"`
}

// ActionOpCommand holds the remote-command parameters for an ActionOperation.
type ActionOpCommand struct {
	Type       int64  `json:"type,string"`
	ScriptID   string `json:"scriptid"`
	ExecuteOn  int64  `json:"execute_on,string"`
	Port       string `json:"port"`
	AuthType   int64  `json:"authtype,string"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	PublicKey  string `json:"publickey"`
	PrivateKey string `json:"privatekey"`
	Command    string `json:"command"`
}

// ActionOpCommandHost is a host target for a remote-command operation.
// HostID "0" means "current host" (the host that triggered the event).
type ActionOpCommandHost struct {
	HostID string `json:"hostid"`
}

// ActionOpCommandGroup is a host-group target for a remote-command operation.
type ActionOpCommandGroup struct {
	GroupID string `json:"groupid"`
}

// --- params helpers ---

func actionWriteParams(a Action) map[string]any {
	return map[string]any{
		"name":                a.Name,
		"status":              a.Status,
		"esc_period":          a.EscPeriod,
		"pause_suppressed":    a.PauseSuppressed,
		"notify_if_canceled":  a.NotifyIfCanceled,
		"filter":              filterWriteParams(a.Filter),
		"operations":          opsWriteParams(a.Operations, true),
		"recovery_operations": opsWriteParams(a.RecoveryOperations, false),
		"update_operations":   opsWriteParams(a.UpdateOperations, false),
	}
}

func filterWriteParams(f ActionFilter) map[string]any {
	customExpr := f.EvalType == 3
	conditions := make([]map[string]any, len(f.Conditions))
	for i, c := range f.Conditions {
		cond := map[string]any{
			"conditiontype": c.ConditionType,
			"operator":      c.Operator,
			"value":         c.Value,
		}
		if c.Value2 != "" {
			cond["value2"] = c.Value2
		}
		if customExpr {
			cond["formulaid"] = c.FormulaID
		}
		conditions[i] = cond
	}
	params := map[string]any{
		"evaltype":   f.EvalType,
		"conditions": conditions,
	}
	if customExpr {
		params["formula"] = f.Formula
	}
	return params
}

// opsWriteParams builds operation params. withEscalation includes esc_period,
// esc_step_from, esc_step_to (for operations); recovery/update ops omit these.
func opsWriteParams(ops []ActionOperation, withEscalation bool) []map[string]any {
	params := make([]map[string]any, len(ops))
	for i, op := range ops {
		params[i] = opWriteParams(op, withEscalation)
	}
	return params
}

func opWriteParams(op ActionOperation, withEscalation bool) map[string]any {
	params := map[string]any{
		"operationtype": op.OperationType,
	}
	if withEscalation {
		params["esc_period"] = op.EscPeriod
		params["esc_step_from"] = op.EscStepFrom
		params["esc_step_to"] = op.EscStepTo
	}
	if op.OpMessage != nil {
		msg := map[string]any{
			"default_msg": op.OpMessage.UseDefault,
		}
		if op.OpMessage.MediaTypeID != "" {
			msg["mediatypeid"] = op.OpMessage.MediaTypeID
		}
		if op.OpMessage.Subject != "" {
			msg["subject"] = op.OpMessage.Subject
		}
		if op.OpMessage.Message != "" {
			msg["message"] = op.OpMessage.Message
		}
		params["opmessage"] = msg
		if op.OpMessageGrp != nil {
			grps := make([]map[string]any, len(op.OpMessageGrp))
			for j, g := range op.OpMessageGrp {
				grps[j] = map[string]any{"usrgrpid": g.UserGroupID}
			}
			params["opmessage_grp"] = grps
		}
		if op.OpMessageUsr != nil {
			usrs := make([]map[string]any, len(op.OpMessageUsr))
			for j, u := range op.OpMessageUsr {
				usrs[j] = map[string]any{"userid": u.UserID}
			}
			params["opmessage_usr"] = usrs
		}
	}
	if op.OpCommand != nil {
		cmd := map[string]any{
			"type":       op.OpCommand.Type,
			"execute_on": op.OpCommand.ExecuteOn,
			"authtype":   op.OpCommand.AuthType,
			"command":    op.OpCommand.Command,
		}
		if op.OpCommand.ScriptID != "" {
			cmd["scriptid"] = op.OpCommand.ScriptID
		}
		if op.OpCommand.Port != "" {
			cmd["port"] = op.OpCommand.Port
		}
		if op.OpCommand.Username != "" {
			cmd["username"] = op.OpCommand.Username
		}
		if op.OpCommand.Password != "" {
			cmd["password"] = op.OpCommand.Password
		}
		if op.OpCommand.PublicKey != "" {
			cmd["publickey"] = op.OpCommand.PublicKey
		}
		if op.OpCommand.PrivateKey != "" {
			cmd["privatekey"] = op.OpCommand.PrivateKey
		}
		params["opcommand"] = cmd
		hsts := make([]map[string]any, len(op.OpCommandHst))
		for j, h := range op.OpCommandHst {
			hsts[j] = map[string]any{"hostid": h.HostID}
		}
		params["opcommand_hst"] = hsts
		grps := make([]map[string]any, len(op.OpCommandGrp))
		for j, g := range op.OpCommandGrp {
			grps[j] = map[string]any{"groupid": g.GroupID}
		}
		params["opcommand_grp"] = grps
	}
	return params
}

// --- API functions ---

// ActionCreate creates a new trigger action and returns its ID.
func ActionCreate(ctx context.Context, c Client, a Action) (string, error) {
	params := actionWriteParams(a)
	params["eventsource"] = 0

	result, err := c.Call(ctx, "action.create", params)
	if err != nil {
		return "", err
	}
	var out struct {
		ActionIDs []string `json:"actionids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("action.create: unexpected response: %w", err)
	}
	if len(out.ActionIDs) == 0 {
		return "", fmt.Errorf("action.create: empty actionids in response")
	}
	return out.ActionIDs[0], nil
}

// ActionGet fetches a trigger action by ID. Returns nil if not found.
func ActionGet(ctx context.Context, c Client, id string) (*Action, error) {
	params := map[string]any{
		"actionids":                []string{id},
		"output":                   "extend",
		"selectFilter":             "extend",
		"selectOperations":         "extend",
		"selectRecoveryOperations": "extend",
		"selectUpdateOperations":   "extend",
		"filter":                   map[string]any{"eventsource": []int{0}},
		"limit":                    1,
	}
	result, err := c.Call(ctx, "action.get", params)
	if err != nil {
		return nil, err
	}
	var actions []Action
	if err := json.Unmarshal(result, &actions); err != nil {
		return nil, fmt.Errorf("action.get: unexpected response: %w", err)
	}
	if len(actions) == 0 {
		return nil, nil
	}
	return &actions[0], nil
}

// ActionGetByName fetches trigger actions by name. Returns all matches.
func ActionGetByName(ctx context.Context, c Client, name string) ([]Action, error) {
	params := map[string]any{
		"output":                   "extend",
		"selectFilter":             "extend",
		"selectOperations":         "extend",
		"selectRecoveryOperations": "extend",
		"selectUpdateOperations":   "extend",
		"filter":                   map[string]any{"name": []string{name}, "eventsource": []int{0}},
	}
	result, err := c.Call(ctx, "action.get", params)
	if err != nil {
		return nil, err
	}
	var actions []Action
	if err := json.Unmarshal(result, &actions); err != nil {
		return nil, fmt.Errorf("action.get: unexpected response: %w", err)
	}
	return actions, nil
}

// ActionUpdate updates an existing trigger action.
func ActionUpdate(ctx context.Context, c Client, a Action) error {
	params := actionWriteParams(a)
	params["actionid"] = a.ActionID
	_, err := c.Call(ctx, "action.update", params)
	return err
}

// ActionDelete deletes a trigger action by ID.
func ActionDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "action.delete", []string{id})
	return err
}
