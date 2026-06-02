// Package clienttest provides test doubles for the client package.
package clienttest

import (
	"context"
	"encoding/json"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// TestClient is a test double for client.Client that returns pre-configured
// responses. Either Response or Error must be set before calling Call.
type TestClient struct {
	// Response is marshalled to JSON and returned by Call when Error is nil.
	Response any
	// Error is returned by Call instead of a response when non-nil.
	Error error
	// LastMethod records the method passed to the most recent Call.
	LastMethod string
	// LastParams records the params passed to the most recent Call.
	LastParams any
}

func (f *TestClient) Call(_ context.Context, method string, params any) (json.RawMessage, error) {
	f.LastMethod = method
	f.LastParams = params
	if f.Error != nil {
		return nil, f.Error
	}
	b, err := json.Marshal(f.Response)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (f *TestClient) APIVersion() string { return "7.0.0" }
func (f *TestClient) Tier() client.Tier  { return client.Targeted }
