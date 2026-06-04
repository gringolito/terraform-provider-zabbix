resource "zabbix_user_group" "network_admins" {
  name         = "Network administrators"
  gui_access   = 0
  debug_mode   = 0
  users_status = 0
}
