package client

import "testing"

func TestClassifyTier(t *testing.T) {
	tests := []struct {
		version string
		want    Tier
	}{
		{"7.0.0", Targeted},
		{"7.0.9", Targeted},
		{"7.0.12", Targeted},
		{"7.2.0", Tolerated},
		{"7.2.3", Tolerated},
		{"7.4.0", Tolerated},
		{"7.4.1", Tolerated},
		{"8.0.0", Unsupported},
		{"6.4.0", Unsupported},
		{"7.6.0", Unsupported},
		{"7.1.0", Unsupported},
		{"7.3.0", Unsupported},
		{"", Unsupported},
		{"invalid", Unsupported},
		{"7", Unsupported},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			got := ClassifyTier(tc.version)
			if got != tc.want {
				t.Errorf("ClassifyTier(%q) = %s, want %s", tc.version, got, tc.want)
			}
		})
	}
}

func TestTierString(t *testing.T) {
	if Targeted.String() != "Targeted" {
		t.Errorf("Targeted.String() = %q", Targeted.String())
	}
	if Tolerated.String() != "Tolerated" {
		t.Errorf("Tolerated.String() = %q", Tolerated.String())
	}
	if Unsupported.String() != "Unsupported" {
		t.Errorf("Unsupported.String() = %q", Unsupported.String())
	}
}
