package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// transientDBErrorResponse mimics Zabbix's opaque wrapper around a SQL error
// (e.g. the DBget_maxid_num ids_pkey cold-start race under concurrency).
const transientDBErrorData = "Database error occurred."

// retryTestServer returns an httptest server that answers apiinfo.version, then
// replies with a -32500 "Database error occurred." envelope for the first
// failFirst non-version calls and a success envelope afterwards. It records how
// many non-version calls it received via the returned counter.
func retryTestServer(t *testing.T, failFirst int) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "apiinfo.version" {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": "7.0.0", "id": 1})
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if int(n) <= failFirst {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32500,
					"message": "Application error.",
					"data":    transientDBErrorData,
				},
				"id": 1,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "result": json.RawMessage(`"ok"`), "id": 1,
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// newTestClient builds a client whose backoff never actually sleeps, so retry
// behaviour can be asserted without wall-clock delay.
func newTestClient(t *testing.T, url string) *jsonrpcClient {
	t.Helper()
	c, err := New(t.Context(), url, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	jc, ok := c.(*jsonrpcClient)
	if !ok {
		t.Fatalf("New returned %T, want *jsonrpcClient", c)
	}
	jc.sleep = func(context.Context, time.Duration) error { return nil }
	return jc
}

func TestCall_RetriesTransientDBError(t *testing.T) {
	t.Parallel()
	srv, calls := retryTestServer(t, 2) // fail twice, then succeed
	c := newTestClient(t, srv.URL)

	result, err := c.Call(t.Context(), "host.create", map[string]any{"host": "x"})
	if err != nil {
		t.Fatalf("Call: expected success after retries, got %v", err)
	}
	if string(result) != `"ok"` {
		t.Errorf("result = %s, want \"ok\"", result)
	}
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Errorf("server received %d calls, want 3 (1 initial + 2 retries)", got)
	}
}

func TestCall_DoesNotRetryNonTransientError(t *testing.T) {
	t.Parallel()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "apiinfo.version" {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "result": "7.0.0", "id": 1})
			return
		}
		atomic.AddInt32(&calls, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": -32602, "message": "Invalid params.", "data": "bad"},
			"id":      1,
		})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	_, err := c.Call(t.Context(), "host.get", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server received %d calls, want 1 (no retry for non-transient error)", got)
	}
}

func TestCall_GivesUpAfterMaxRetries(t *testing.T) {
	t.Parallel()
	srv, calls := retryTestServer(t, 1000) // always fail
	c := newTestClient(t, srv.URL)

	_, err := c.Call(t.Context(), "host.create", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	want := int32(c.maxRetries + 1)
	if got := atomic.LoadInt32(calls); got != want {
		t.Errorf("server received %d calls, want %d (1 initial + %d retries)", got, want, c.maxRetries)
	}
}

func TestCall_RespectsContextCancellationDuringBackoff(t *testing.T) {
	t.Parallel()
	srv, _ := retryTestServer(t, 1000) // always fail, forcing a backoff
	c := newRealSleepClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	_, err := c.Call(ctx, "host.create", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context deadline error, got %v", err)
	}
}

// newRealSleepClient builds a client with the real ctx-aware sleeper so the
// cancellation path is exercised end to end.
func newRealSleepClient(t *testing.T, url string) *jsonrpcClient {
	t.Helper()
	c, err := New(t.Context(), url, "tok")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	jc, ok := c.(*jsonrpcClient)
	if !ok {
		t.Fatalf("New returned %T, want *jsonrpcClient", c)
	}
	return jc
}

func TestIsTransientDBError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"transient db error", &RPCError{Code: -32500, Message: "Application error.", Data: json.RawMessage(`"Database error occurred."`)}, true},
		{"other -32500", &RPCError{Code: -32500, Message: "Application error.", Data: json.RawMessage(`"Something else."`)}, false},
		{"invalid params", &RPCError{Code: -32602, Message: "Invalid params.", Data: json.RawMessage(`"bad"`)}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isTransientDBError(tc.err); got != tc.want {
				t.Errorf("isTransientDBError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
