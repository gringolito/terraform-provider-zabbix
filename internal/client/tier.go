package client

import "strings"

// Tier represents the compatibility classification of a Zabbix server version.
type Tier int

const (
	Targeted    Tier = iota // 7.0.x — fully supported
	Tolerated               // 7.2.x, 7.4.x — best-effort support
	Unsupported             // everything else
)

func (t Tier) String() string {
	switch t {
	case Targeted:
		return "Targeted"
	case Tolerated:
		return "Tolerated"
	default:
		return "Unsupported"
	}
}

// ClassifyTier maps a Zabbix version string to a Tier. Expects "major.minor.patch"
// format; anything unparseable is Unsupported.
func ClassifyTier(version string) Tier {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return Unsupported
	}
	major := parts[0]
	minor := parts[1]

	switch {
	case major == "7" && minor == "0":
		return Targeted
	case major == "7" && (minor == "2" || minor == "4"):
		return Tolerated
	default:
		return Unsupported
	}
}
