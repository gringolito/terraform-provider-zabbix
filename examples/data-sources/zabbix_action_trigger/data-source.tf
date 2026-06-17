# Look up a trigger action by name.
data "zabbix_action_trigger" "by_name" {
  name = "Notify administrators"
}

# Look up a trigger action by ID.
data "zabbix_action_trigger" "by_id" {
  id = "12"
}
