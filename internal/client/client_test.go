package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

// rpcMethod is a minimal struct used to dispatch handlers by RPC method name.
type rpcMethod struct {
	Method string `json:"method"`
}

func versionHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"jsonrpc": "2.0",
			"result":  version,
			"id":      1,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func rpcServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// captureHandler records the last request and delegates to inner.
type captureHandler struct {
	last  *http.Request
	body  []byte
	inner http.HandlerFunc
}

func (c *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.last = r
	body, _ := io.ReadAll(r.Body)
	c.body = body
	c.inner(w, r)
}

func TestNew_DetectsVersion(t *testing.T) {
	srv := rpcServer(t, versionHandler("7.0.3"))
	c, err := client.New(t.Context(), srv.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.APIVersion() != "7.0.3" {
		t.Errorf("APIVersion = %q, want %q", c.APIVersion(), "7.0.3")
	}
}

func TestCall_SendsBearerAuth(t *testing.T) {
	ch := &captureHandler{}
	ch.inner = versionHandler("7.0.0")
	srv := rpcServer(t, ch.ServeHTTP)

	_, err := client.New(t.Context(), srv.URL, "my-secret-token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got := ch.last.Header.Get("Authorization")
	if got != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-secret-token")
	}
}

func TestCall_SuccessResponse(t *testing.T) {
	srv := rpcServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req rpcMethod
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if req.Method == "apiinfo.version" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "result": "7.0.0", "id": 1,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "result": map[string]any{"hostids": []string{"42"}}, "id": 1,
		})
	})

	c, err := client.New(t.Context(), srv.URL, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	result, err := c.Call(t.Context(), "host.create", map[string]any{"host": "test"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := out["hostids"]; !ok {
		t.Errorf("expected hostids in result, got %v", out)
	}
}

func TestCall_ErrorEnvelopePreservedVerbatim(t *testing.T) {
	srv := rpcServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req rpcMethod
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if req.Method == "apiinfo.version" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "result": "7.0.0", "id": 1,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error": map[string]any{
				"code":    -32602,
				"message": "Invalid params.",
				"data":    "No permissions to referred object or it does not exist!",
			},
			"id": 1,
		})
	})

	c, err := client.New(t.Context(), srv.URL, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Call(t.Context(), "host.get", map[string]any{"hostids": "999"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rpcErr, ok := err.(*client.RPCError)
	if !ok {
		t.Fatalf("expected *client.RPCError, got %T: %v", err, err)
	}
	if rpcErr.Code != -32602 {
		t.Errorf("Code = %d, want -32602", rpcErr.Code)
	}
	if rpcErr.Message != "Invalid params." {
		t.Errorf("Message = %q, want %q", rpcErr.Message, "Invalid params.")
	}
	if !strings.Contains(string(rpcErr.Data), "No permissions") {
		t.Errorf("Data = %s, want to contain 'No permissions'", rpcErr.Data)
	}
}

func TestCall_MalformedJSON(t *testing.T) {
	srv := rpcServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req rpcMethod
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if req.Method == "apiinfo.version" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "result": "7.0.0", "id": 1,
			})
			return
		}
		_, _ = w.Write([]byte("{not valid json"))
	})

	c, err := client.New(t.Context(), srv.URL, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Call(t.Context(), "host.get", nil)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestCall_HTTPNon2xx(t *testing.T) {
	srv := rpcServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req rpcMethod
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Method == "apiinfo.version" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "result": "7.0.0", "id": 1,
			})
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	c, err := client.New(t.Context(), srv.URL, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Call(t.Context(), "host.get", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status code, got: %v", err)
	}
}
