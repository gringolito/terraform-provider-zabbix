package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is the interface for making Zabbix API JSON-RPC calls.
type Client interface {
	// Call invokes a Zabbix API method and returns the raw result JSON.
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)
	// APIVersion returns the Zabbix server version string detected at construction.
	APIVersion() string
	// Tier returns the compatibility tier for the connected Zabbix server.
	Tier() Tier
}

type jsonrpcClient struct {
	url     string
	token   string
	version string
	http    *http.Client
}

// New constructs a Client, connects to the Zabbix API at url, detects the
// server version via apiinfo.version, and returns an error if the server is
// unreachable or returns a malformed response.
func New(ctx context.Context, url, token string) (Client, error) {
	c := &jsonrpcClient{
		url:   strings.TrimRight(url, "/") + "/api_jsonrpc.php",
		token: token,
		http:  &http.Client{},
	}
	result, err := c.Call(ctx, "apiinfo.version", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("zabbix version detection: %w", err)
	}
	if err := json.Unmarshal(result, &c.version); err != nil {
		return nil, fmt.Errorf("zabbix version detection: unexpected response: %w", err)
	}
	return c, nil
}

func (c *jsonrpcClient) APIVersion() string { return c.version }
func (c *jsonrpcClient) Tier() Tier         { return ClassifyTier(c.version) }

func (c *jsonrpcClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, body)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}
