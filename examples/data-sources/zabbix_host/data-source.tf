# Look up a host by its ID
data "zabbix_host" "by_id" {
  id = "42"
}

# Look up a host by its technical name
data "zabbix_host" "by_name" {
  host = "web-srv-01"
}
