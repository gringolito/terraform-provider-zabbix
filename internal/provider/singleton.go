package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	authenticationSingletonDeleteSummary = "zabbix_authentication was reset to defaults, not deleted"
	authenticationSingletonDeleteDetail  = "zabbix_authentication is a Zabbix global singleton that cannot be deleted via the API. " +
		"Terraform has reset it to Zabbix 7.0 documented defaults. " +
		"Keep the resource block in your configuration to continue managing it."
)

// singletonWarnOnDelete adds a visible Terraform diagnostic warning and logs
// the fact that zabbix_authentication was reset to defaults rather than deleted.
// Both the diagnostic (visible in terraform destroy output) and the log entry
// are emitted, as required by ADR-0014.
func singletonWarnOnDelete(ctx context.Context, diags *diag.Diagnostics) {
	diags.AddWarning(authenticationSingletonDeleteSummary, authenticationSingletonDeleteDetail)
	tflog.Warn(ctx, authenticationSingletonDeleteSummary, map[string]any{
		"detail": authenticationSingletonDeleteDetail,
	})
}
