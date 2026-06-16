package provider

import "github.com/gringolito/terraform-provider-zabbix/internal/client"

// ResetAuthenticationSingletonForTesting clears the process-wide singleton
// registry. Call via t.Cleanup in unit tests to prevent state leaking between
// tests.
func ResetAuthenticationSingletonForTesting() {
	authSingletonMu.Lock()
	authSingletonSet = make(map[client.Client]struct{})
	authSingletonMu.Unlock()
}

// RegisterAuthenticationSingletonForTesting inserts c into the singleton
// registry, simulating a prior successful Create. Used to set up the precondition
// for tests without needing a full plan-backed Create call.
func RegisterAuthenticationSingletonForTesting(c client.Client) {
	authSingletonMu.Lock()
	authSingletonSet[c] = struct{}{}
	authSingletonMu.Unlock()
}

// IsAuthenticationSingletonRegisteredForTesting reports whether c is currently
// in the singleton registry. Used to assert the Delete unregistration path.
func IsAuthenticationSingletonRegisteredForTesting(c client.Client) bool {
	authSingletonMu.Lock()
	defer authSingletonMu.Unlock()
	_, ok := authSingletonSet[c]
	return ok
}
