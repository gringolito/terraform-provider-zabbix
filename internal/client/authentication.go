package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Authentication represents the Zabbix global authentication configuration.
// Zabbix 7.0 returns integer fields as JSON strings.
// Reference: https://www.zabbix.com/documentation/7.0/en/manual/api/reference/authentication/object
type Authentication struct {
	AuthenticationType   int64  `json:"authentication_type,string"`
	HTTPAuthEnabled      int64  `json:"http_auth_enabled,string"`
	HTTPLoginForm        int64  `json:"http_login_form,string"`
	HTTPStripDomains     string `json:"http_strip_domains"`
	HTTPCaseSensitive    int64  `json:"http_case_sensitive,string"`
	LDAPAuthEnabled      int64  `json:"ldap_auth_enabled,string"`
	LDAPCaseSensitive    int64  `json:"ldap_case_sensitive,string"`
	LDAPUserDirectoryID  string `json:"ldap_userdirectoryid"`
	SAMLAuthEnabled      int64  `json:"saml_auth_enabled,string"`
	SAMLCaseSensitive    int64  `json:"saml_case_sensitive,string"`
	PasswdMinLength      int64  `json:"passwd_min_length,string"`
	PasswdCheckRules     int64  `json:"passwd_check_rules,string"`
	JITProvisionInterval string `json:"jit_provision_interval"`
	SAMLJITStatus        int64  `json:"saml_jit_status,string"`
	LDAPJITStatus        int64  `json:"ldap_jit_status,string"`
	DisabledUsrgrpID     string `json:"disabled_usrgrpid"`
	MFAStatus            int64  `json:"mfa_status,string"`
	MFAID                string `json:"mfaid"`
}

// AuthenticationGet retrieves the global authentication configuration.
// Returns a single object (not an array) per the Zabbix 7.0 API.
func AuthenticationGet(ctx context.Context, c Client) (*Authentication, error) {
	result, err := c.Call(ctx, "authentication.get", map[string]any{
		"output": "extend",
	})
	if err != nil {
		return nil, err
	}
	var auth Authentication
	if err := json.Unmarshal(result, &auth); err != nil {
		return nil, fmt.Errorf("authentication.get: unexpected response: %w", err)
	}
	return &auth, nil
}

// AuthenticationUpdate updates the global authentication configuration.
// Zabbix 7.0 documented defaults for reset-on-delete are encoded in the
// authentication resource Delete method — see authentication_resource.go.
func AuthenticationUpdate(ctx context.Context, c Client, auth Authentication) error {
	params := map[string]any{
		"authentication_type":    auth.AuthenticationType,
		"http_auth_enabled":      auth.HTTPAuthEnabled,
		"http_login_form":        auth.HTTPLoginForm,
		"http_strip_domains":     auth.HTTPStripDomains,
		"http_case_sensitive":    auth.HTTPCaseSensitive,
		"ldap_auth_enabled":      auth.LDAPAuthEnabled,
		"ldap_case_sensitive":    auth.LDAPCaseSensitive,
		"ldap_userdirectoryid":   auth.LDAPUserDirectoryID,
		"saml_auth_enabled":      auth.SAMLAuthEnabled,
		"saml_case_sensitive":    auth.SAMLCaseSensitive,
		"passwd_min_length":      auth.PasswdMinLength,
		"passwd_check_rules":     auth.PasswdCheckRules,
		"jit_provision_interval": auth.JITProvisionInterval,
		"saml_jit_status":        auth.SAMLJITStatus,
		"ldap_jit_status":        auth.LDAPJITStatus,
		"disabled_usrgrpid":      auth.DisabledUsrgrpID,
		"mfa_status":             auth.MFAStatus,
		"mfaid":                  auth.MFAID,
	}
	_, err := c.Call(ctx, "authentication.update", params)
	return err
}
