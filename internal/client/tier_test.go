package client_test

import (
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

func TestClassifyTier(t *testing.T) {
	tests := []struct {
		version string
		want    client.Tier
	}{
		{"7.0.0", client.Targeted},
		{"7.0.9", client.Targeted},
		{"7.0.12", client.Targeted},
		{"7.2.0", client.Tolerated},
		{"7.2.3", client.Tolerated},
		{"7.4.0", client.Tolerated},
		{"7.4.1", client.Tolerated},
		{"8.0.0", client.Unsupported},
		{"6.4.0", client.Unsupported},
		{"7.6.0", client.Unsupported},
		{"7.1.0", client.Unsupported},
		{"7.3.0", client.Unsupported},
		{"", client.Unsupported},
		{"invalid", client.Unsupported},
		{"7", client.Unsupported},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			got := client.ClassifyTier(tc.version)
			if got != tc.want {
				t.Errorf("ClassifyTier(%q) = %s, want %s", tc.version, got, tc.want)
			}
		})
	}
}

func TestTierString(t *testing.T) {
	if client.Targeted.String() != "Targeted" {
		t.Errorf("Targeted.String() = %q", client.Targeted.String())
	}
	if client.Tolerated.String() != "Tolerated" {
		t.Errorf("Tolerated.String() = %q", client.Tolerated.String())
	}
	if client.Unsupported.String() != "Unsupported" {
		t.Errorf("Unsupported.String() = %q", client.Unsupported.String())
	}
}
