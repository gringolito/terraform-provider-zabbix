# Lookup by ID
data "zabbix_trigger" "by_id" {
  id = "12345"
}

# Lookup by description scoped to a host
data "zabbix_trigger" "by_host" {
  description = "High CPU utilization"
  host_id     = "10084"
}

# Lookup by description scoped to a template
data "zabbix_trigger" "by_template" {
  description = "High CPU utilization"
  template_id = "10085"
}
