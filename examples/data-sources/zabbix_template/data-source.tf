# Look up by ID
data "zabbix_template" "by_id" {
  id = "10"
}

# Look up by technical name
data "zabbix_template" "by_host" {
  host = "Linux by Zabbix agent"
}
