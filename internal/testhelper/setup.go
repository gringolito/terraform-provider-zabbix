package testhelper

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Config holds the connection details and name-prefix for an acceptance test run.
type Config struct {
	URL        string
	Token      string
	NamePrefix string
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Setup skips the test if TF_ACC is not set, reads ZABBIX_URL and ZABBIX_TOKEN
// from the environment, and returns a Config with a unique name prefix of the
// form "tf-acc-<test>-<rand6>" where <test> is a sanitised version of t.Name().
// A t.Cleanup hook is registered (currently a no-op; per-resource teardown is
// handled by the test itself via Terraform destroy).
func Setup(t *testing.T) *Config {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}

	url := os.Getenv("ZABBIX_URL")
	if url == "" {
		t.Fatal("ZABBIX_URL must be set for acceptance tests")
	}

	token := os.Getenv("ZABBIX_TOKEN")
	if token == "" {
		t.Fatal("ZABBIX_TOKEN must be set for acceptance tests")
	}

	slug := nonAlnum.ReplaceAllString(strings.ToLower(t.Name()), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 24 {
		slug = slug[:24]
	}
	prefix := fmt.Sprintf("tf-acc-%s-%s", slug, randHex(6))

	t.Cleanup(func() {})

	return &Config{
		URL:        url,
		Token:      token,
		NamePrefix: prefix,
	}
}

func randHex(n int) string {
	const chars = "abcdef0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
