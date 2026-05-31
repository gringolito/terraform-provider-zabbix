package client

import (
	"encoding/json"
	"fmt"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      int    `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
	ID      int             `json:"id"`
}

// RPCError represents a Zabbix JSON-RPC error envelope, preserved verbatim.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("zabbix api error %d: %s (data: %s)", e.Code, e.Message, e.Data)
}
