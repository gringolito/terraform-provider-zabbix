resource "zabbix_user_group" "network_admins" {
  name         = "Network administrators"
  gui_access   = "system_default"
  debug_mode   = "disabled"
  users_status = "enabled"
}
