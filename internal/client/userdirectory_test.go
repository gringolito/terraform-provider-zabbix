package client_test

import (
	"net/http"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
)

var ldapUD = client.UserDirectory{
	IDPType:         1,
	Name:            "test-ldap",
	Host:            "ldap.example.com",
	Port:            389,
	BaseDN:          "dc=example,dc=com",
	SearchAttribute: "uid",
}

var samlUD = client.UserDirectory{
	IDPType:           2,
	Name:              "test-saml",
	IDPEntityID:       "http://idp.example.com/metadata",
	SPEntityID:        "zabbix",
	UsernameAttribute: "uid",
}

func TestUserDirectoryCreate_LDAP_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.create": rpcOK(t, map[string]any{"userdirectoryids": []string{"10"}}),
	})
	id, err := client.UserDirectoryCreate(t.Context(), c, ldapUD)
	if err != nil {
		t.Fatalf("UserDirectoryCreate: %v", err)
	}
	if id != "10" {
		t.Errorf("id = %q, want %q", id, "10")
	}
}

func TestUserDirectoryCreate_SAML_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.create": rpcOK(t, map[string]any{"userdirectoryids": []string{"11"}}),
	})
	id, err := client.UserDirectoryCreate(t.Context(), c, samlUD)
	if err != nil {
		t.Fatalf("UserDirectoryCreate: %v", err)
	}
	if id != "11" {
		t.Errorf("id = %q, want %q", id, "11")
	}
}

func TestUserDirectoryCreate_ErrorEnvelope(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.create": rpcErr(t, -32602, "Invalid params."),
	})
	_, err := client.UserDirectoryCreate(t.Context(), c, ldapUD)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserDirectoryGet_Found(t *testing.T) {
	resp := map[string]any{
		"userdirectoryid":  "10",
		"idp_type":         "1",
		"name":             "test-ldap",
		"description":      "",
		"provision_status": "0",
		"group_name":       "",
		"user_username":    "",
		"user_lastname":    "",
		"host":             "ldap.example.com",
		"port":             "389",
		"base_dn":          "dc=example,dc=com",
		"search_attribute": "uid",
		"bind_dn":          "",
		"start_tls":        "0",
		"search_filter":    "",
		"group_basedn":     "",
		"group_member":     "",
		"group_filter":     "",
		"group_membership": "",
		"user_ref_attr":    "",
		"provision_groups": []any{},
		"provision_media":  []any{},
	}
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.get": rpcOK(t, []any{resp}),
	})
	ud, err := client.UserDirectoryGet(t.Context(), c, "10")
	if err != nil {
		t.Fatalf("UserDirectoryGet: %v", err)
	}
	if ud == nil {
		t.Fatal("expected non-nil result")
	}
	if ud.Name != "test-ldap" {
		t.Errorf("Name = %q, want %q", ud.Name, "test-ldap")
	}
	if ud.Port != 389 {
		t.Errorf("Port = %d, want 389", ud.Port)
	}
}

func TestUserDirectoryGet_NotFound(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.get": rpcOK(t, []any{}),
	})
	ud, err := client.UserDirectoryGet(t.Context(), c, "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ud != nil {
		t.Fatalf("expected nil, got %+v", ud)
	}
}

func TestUserDirectoryGetByName_ReturnsMultiple(t *testing.T) {
	base := map[string]any{
		"description": "", "provision_status": "0", "group_name": "", "user_username": "",
		"user_lastname": "", "host": "a", "port": "389", "base_dn": "dc=a",
		"search_attribute": "uid", "bind_dn": "", "start_tls": "0", "search_filter": "",
		"group_basedn": "", "group_member": "", "group_filter": "", "group_membership": "",
		"user_ref_attr": "", "provision_groups": []any{}, "provision_media": []any{},
	}
	r1 := map[string]any{"userdirectoryid": "1", "idp_type": "1", "name": "ldap"}
	r2 := map[string]any{"userdirectoryid": "2", "idp_type": "1", "name": "ldap"}
	for k, v := range base {
		r1[k] = v
		r2[k] = v
	}
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.get": rpcOK(t, []any{r1, r2}),
	})
	dirs, err := client.UserDirectoryGetByName(t.Context(), c, "ldap", 1)
	if err != nil {
		t.Fatalf("UserDirectoryGetByName: %v", err)
	}
	if len(dirs) != 2 {
		t.Errorf("len = %d, want 2", len(dirs))
	}
}

func TestUserDirectoryUpdate_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.update": rpcOK(t, map[string]any{"userdirectoryids": []string{"10"}}),
	})
	ud := ldapUD
	ud.ID = "10"
	if err := client.UserDirectoryUpdate(t.Context(), c, ud); err != nil {
		t.Fatalf("UserDirectoryUpdate: %v", err)
	}
}

func TestUserDirectoryDelete_Success(t *testing.T) {
	c := newTestClient(t, map[string]http.HandlerFunc{
		"userdirectory.delete": rpcOK(t, map[string]any{"userdirectoryids": []string{"10"}}),
	})
	if err := client.UserDirectoryDelete(t.Context(), c, "10"); err != nil {
		t.Fatalf("UserDirectoryDelete: %v", err)
	}
}
