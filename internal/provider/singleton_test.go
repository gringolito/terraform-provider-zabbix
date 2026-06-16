package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestSingletonWarnOnDelete_AddsDiagnosticWarning(t *testing.T) {
	var diags diag.Diagnostics
	singletonWarnOnDelete(context.Background(), &diags)
	if diags.WarningsCount() != 1 {
		t.Errorf("expected 1 warning, got %d", diags.WarningsCount())
	}
}

func TestSingletonWarnOnDelete_NoErrors(t *testing.T) {
	var diags diag.Diagnostics
	singletonWarnOnDelete(context.Background(), &diags)
	if diags.HasError() {
		t.Error("singletonWarnOnDelete must not add errors, only warnings")
	}
}

func TestSingletonWarnOnDelete_SummaryIsNonEmpty(t *testing.T) {
	var diags diag.Diagnostics
	singletonWarnOnDelete(context.Background(), &diags)
	warns := diags.Warnings()
	if len(warns) == 0 {
		t.Fatal("expected at least one warning")
	}
	if warns[0].Summary() == "" {
		t.Error("warning summary must be non-empty")
	}
}
