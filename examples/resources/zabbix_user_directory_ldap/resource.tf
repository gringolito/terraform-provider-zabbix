resource "zabbix_user_directory_ldap" "corporate" {
  name             = "Corporate Active Directory"
  host             = "ad.example.com"
  port             = 389
  base_dn          = "DC=example,DC=com"
  search_attribute = "sAMAccountName"
  bind_dn          = "CN=svc-zabbix,OU=Service Accounts,DC=example,DC=com"
  bind_password    = var.ldap_bind_password
  start_tls        = "disabled"

  group_base_dn = "OU=Groups,DC=example,DC=com"
  group_name    = "CN"
  group_member  = "member"
  user_ref_attr = "CN"

  user_username = "givenName"
  user_lastname = "sn"

  provision_status = "enabled"

  provision_groups = [
    {
      name           = "Zabbix Admins"
      role_id        = zabbix_role.admin.id
      user_group_ids = [zabbix_user_group.admins.id]
    },
    {
      name           = "*"
      role_id        = zabbix_role.viewer.id
      user_group_ids = [zabbix_user_group.all_users.id]
    },
  ]

  provision_media = [
    {
      name          = "email"
      media_type_id = zabbix_media_type_email.smtp.id
      attribute     = "mail"
      active        = "enabled"
      severity      = 63
      period        = "1-7,00:00-24:00"
    },
  ]
}
