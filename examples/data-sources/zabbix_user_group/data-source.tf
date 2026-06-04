# Look up by ID
data "zabbix_user_group" "by_id" {
  id = "7"
}

# Look up by name
data "zabbix_user_group" "by_name" {
  name = "Network administrators"
}
