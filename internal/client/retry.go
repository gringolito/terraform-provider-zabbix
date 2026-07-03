package client

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"
)

// Retry tuning for transient Zabbix database errors. Zabbix serialises entity-id
// allocation through the single-row-per-entity `ids` table via
// `SELECT nextid ... FOR UPDATE`. On a fresh database the seed row for an entity
// type does not exist yet, so two concurrent creates of the same type can both
// try to INSERT it, and one loses with a duplicate-key violation. Zabbix reports
// this (and lock-wait/deadlock aborts) opaquely as JSON-RPC error -32500 with
// data "Database error occurred.". These aborts roll back the whole API
// transaction, so the call leaves no partial state and is safe to replay.
const (
	defaultMaxRetries  = 6
	retryBaseBackoff   = 50 * time.Millisecond
	retryMaxBackoff    = 2 * time.Second
	transientDBErrText = "Database error occurred."
)

// isTransientDBError reports whether err is a Zabbix -32500 error whose data
// signals an underlying database failure — the retryable class described above.
func isTransientDBError(err error) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	return rpcErr.Code == -32500 && strings.Contains(string(rpcErr.Data), transientDBErrText)
}

// backoffFor returns the delay before retry attempt n (n >= 1): exponential
// growth from retryBaseBackoff, capped at retryMaxBackoff, with equal jitter so
// concurrent callers that collided do not retry in lockstep.
func backoffFor(n int) time.Duration {
	d := retryBaseBackoff << (n - 1)
	if d <= 0 || d > retryMaxBackoff {
		d = retryMaxBackoff
	}
	half := d / 2
	return half + time.Duration(rand.Int63n(int64(half)+1))
}

// sleepWithContext waits for d or until ctx is done, returning ctx.Err() if the
// context is cancelled first so retries abort promptly on deadline/cancel.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
