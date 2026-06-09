package client

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	IDPTypeLDAP int64 = 1
	IDPTypeSAML int64 = 2
)

type ProvisionUserGroup struct {
	ID string `json:"usrgrpid"`
}

type ProvisionGroup struct {
	Name       string               `json:"name"`
	RoleID     string               `json:"roleid"`
	UserGroups []ProvisionUserGroup `json:"user_groups"`
}

type ProvisionMedia struct {
	MediaID     string `json:"userdirectory_mediaid,omitempty"`
	Name        string `json:"name"`
	MediaTypeID string `json:"mediatypeid"`
	Attribute   string `json:"attribute"`
	Active      int64  `json:"active,string"`
	Severity    int64  `json:"severity,string"`
	Period      string `json:"period"`
}

// UserDirectory represents a Zabbix user directory (LDAP or SAML).
// Zabbix 7.0 returns integer fields as JSON strings.
// BindPassword is write-only and never returned by the API.
type UserDirectory struct {
	ID              string `json:"userdirectoryid,omitempty"`
	IDPType         int64  `json:"idp_type,string"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ProvisionStatus int64  `json:"provision_status,string"`
	GroupName       string `json:"group_name"`
	UserUsername    string `json:"user_username"`
	UserLastname    string `json:"user_lastname"`
	// LDAP-only
	Host            string `json:"host"`
	Port            int64  `json:"port,string"`
	BaseDN          string `json:"base_dn"`
	SearchAttribute string `json:"search_attribute"`
	BindDN          string `json:"bind_dn"`
	BindPassword    string `json:"bind_password,omitempty"`
	StartTLS        int64  `json:"start_tls,string"`
	SearchFilter    string `json:"search_filter"`
	GroupBaseDN     string `json:"group_basedn"`
	GroupMember     string `json:"group_member"`
	GroupFilter     string `json:"group_filter"`
	GroupMembership string `json:"group_membership"`
	UserRefAttr     string `json:"user_ref_attr"`
	// SAML-only
	IDPEntityID         string `json:"idp_entityid"`
	SPEntityID          string `json:"sp_entityid"`
	UsernameAttribute   string `json:"username_attribute"`
	SSOURL              string `json:"sso_url"`
	SLOURL              string `json:"slo_url"`
	NameIDFormat        string `json:"nameid_format"`
	SignMessages        int64  `json:"sign_messages,string"`
	SignAssertions      int64  `json:"sign_assertions,string"`
	SignAuthnRequests   int64  `json:"sign_authn_requests,string"`
	SignLogoutRequests  int64  `json:"sign_logout_requests,string"`
	SignLogoutResponses int64  `json:"sign_logout_responses,string"`
	EncryptNameID       int64  `json:"encrypt_nameid,string"`
	EncryptAssertions   int64  `json:"encrypt_assertions,string"`
	SCIMStatus          int64  `json:"scim_status,string"`
	// Always present
	ProvisionGroups []ProvisionGroup `json:"provision_groups"`
	ProvisionMedia  []ProvisionMedia `json:"provision_media"`
}

func UserDirectoryCreate(ctx context.Context, c Client, ud UserDirectory) (string, error) {
	params := udCommonParams(ud)
	if ud.IDPType == IDPTypeLDAP {
		udAddLDAPParams(params, ud)
	} else {
		udAddSAMLParams(params, ud)
	}
	result, err := c.Call(ctx, "userdirectory.create", params)
	if err != nil {
		return "", err
	}
	var out struct {
		IDs []string `json:"userdirectoryids"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		return "", fmt.Errorf("userdirectory.create: unexpected response: %w", err)
	}
	if len(out.IDs) == 0 {
		return "", fmt.Errorf("userdirectory.create: empty userdirectoryids in response")
	}
	return out.IDs[0], nil
}

func UserDirectoryGet(ctx context.Context, c Client, id string) (*UserDirectory, error) {
	result, err := c.Call(ctx, "userdirectory.get", map[string]any{
		"userdirectoryids":      []string{id},
		"output":                "extend",
		"selectProvisionGroups": "extend",
		"selectProvisionMedia":  "extend",
		"limit":                 1,
	})
	if err != nil {
		return nil, err
	}
	var dirs []UserDirectory
	if err := json.Unmarshal(result, &dirs); err != nil {
		return nil, fmt.Errorf("userdirectory.get: unexpected response: %w", err)
	}
	if len(dirs) == 0 {
		return nil, nil
	}
	return &dirs[0], nil
}

func UserDirectoryGetByName(ctx context.Context, c Client, name string, idpType int64) ([]UserDirectory, error) {
	// userdirectory.get does not support filter.name; use search + client-side filter for idp_type and exact name
	result, err := c.Call(ctx, "userdirectory.get", map[string]any{
		"search":                map[string]any{"name": name},
		"output":                "extend",
		"selectProvisionGroups": "extend",
		"selectProvisionMedia":  "extend",
	})
	if err != nil {
		return nil, err
	}
	var all []UserDirectory
	if err := json.Unmarshal(result, &all); err != nil {
		return nil, fmt.Errorf("userdirectory.get: unexpected response: %w", err)
	}
	var dirs []UserDirectory
	for _, d := range all {
		if d.Name == name && d.IDPType == idpType {
			dirs = append(dirs, d)
		}
	}
	return dirs, nil
}

func UserDirectoryUpdate(ctx context.Context, c Client, ud UserDirectory) error {
	params := udCommonParams(ud)
	params["userdirectoryid"] = ud.ID
	if ud.IDPType == IDPTypeLDAP {
		udAddLDAPParams(params, ud)
	} else {
		udAddSAMLParams(params, ud)
	}
	_, err := c.Call(ctx, "userdirectory.update", params)
	return err
}

func UserDirectoryDelete(ctx context.Context, c Client, id string) error {
	_, err := c.Call(ctx, "userdirectory.delete", []string{id})
	return err
}

func udCommonParams(ud UserDirectory) map[string]any {
	return map[string]any{
		"idp_type":         ud.IDPType,
		"name":             ud.Name,
		"description":      ud.Description,
		"provision_status": ud.ProvisionStatus,
		"group_name":       ud.GroupName,
		"user_username":    ud.UserUsername,
		"user_lastname":    ud.UserLastname,
		"provision_groups": provisionGroupsToParams(ud.ProvisionGroups),
		"provision_media":  provisionMediaToParams(ud.ProvisionMedia),
	}
}

func udAddLDAPParams(p map[string]any, ud UserDirectory) {
	p["host"] = ud.Host
	p["port"] = ud.Port
	p["base_dn"] = ud.BaseDN
	p["search_attribute"] = ud.SearchAttribute
	p["bind_dn"] = ud.BindDN
	p["start_tls"] = ud.StartTLS
	p["search_filter"] = ud.SearchFilter
	p["group_basedn"] = ud.GroupBaseDN
	p["group_member"] = ud.GroupMember
	p["group_filter"] = ud.GroupFilter
	p["group_membership"] = ud.GroupMembership
	p["user_ref_attr"] = ud.UserRefAttr
	if ud.BindPassword != "" {
		p["bind_password"] = ud.BindPassword
	}
}

func udAddSAMLParams(p map[string]any, ud UserDirectory) {
	p["idp_entityid"] = ud.IDPEntityID
	p["sp_entityid"] = ud.SPEntityID
	p["username_attribute"] = ud.UsernameAttribute
	p["sso_url"] = ud.SSOURL
	p["slo_url"] = ud.SLOURL
	p["nameid_format"] = ud.NameIDFormat
	p["sign_messages"] = ud.SignMessages
	p["sign_assertions"] = ud.SignAssertions
	p["sign_authn_requests"] = ud.SignAuthnRequests
	p["sign_logout_requests"] = ud.SignLogoutRequests
	p["sign_logout_responses"] = ud.SignLogoutResponses
	p["encrypt_nameid"] = ud.EncryptNameID
	p["encrypt_assertions"] = ud.EncryptAssertions
	p["scim_status"] = ud.SCIMStatus
}

func provisionGroupsToParams(groups []ProvisionGroup) []map[string]any {
	if len(groups) == 0 {
		return []map[string]any{}
	}
	result := make([]map[string]any, len(groups))
	for i, g := range groups {
		ugs := make([]map[string]any, len(g.UserGroups))
		for j, ug := range g.UserGroups {
			ugs[j] = map[string]any{"usrgrpid": ug.ID}
		}
		result[i] = map[string]any{
			"name":        g.Name,
			"roleid":      g.RoleID,
			"user_groups": ugs,
		}
	}
	return result
}

func provisionMediaToParams(media []ProvisionMedia) []map[string]any {
	if len(media) == 0 {
		return []map[string]any{}
	}
	result := make([]map[string]any, len(media))
	for i, m := range media {
		result[i] = map[string]any{
			"name":        m.Name,
			"mediatypeid": m.MediaTypeID,
			"attribute":   m.Attribute,
			"active":      m.Active,
			"severity":    m.Severity,
			"period":      m.Period,
		}
	}
	return result
}
