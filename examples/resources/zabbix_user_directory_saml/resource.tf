resource "zabbix_user_directory_saml" "okta" {
  name               = "Okta SSO"
  idp_entityid       = "http://www.okta.com/exkABCDEF1234567890"
  sp_entityid        = "zabbix"
  username_attribute = "email"
  sso_url            = "https://example.okta.com/app/zabbix/exkABCDEF1234567890/sso/saml"
  slo_url            = "https://example.okta.com/app/zabbix/exkABCDEF1234567890/slo/saml"

  sign_messages   = "enabled"
  sign_assertions = "enabled"
  encrypt_nameid  = "disabled"
  scim_status     = "enabled"

  group_name       = "groups"
  user_username    = "firstName"
  user_lastname    = "lastName"
  provision_status = "enabled"

  provision_groups = [
    {
      name           = "zabbix-admins"
      role_id        = zabbix_role.admin.id
      user_group_ids = [zabbix_user_group.admins.id]
    },
  ]
}
