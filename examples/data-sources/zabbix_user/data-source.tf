# Look up a Zabbix user by username
data "zabbix_user" "by_username" {
  username = "Admin"
}

# Look up a Zabbix user by ID
data "zabbix_user" "by_id" {
  id = "1"
}
