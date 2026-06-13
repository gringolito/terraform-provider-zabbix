# Lookup by ID
data "zabbix_item" "by_id" {
  id = "12345"
}

# Lookup by key scoped to a host
data "zabbix_item" "by_host" {
  key     = "system.cpu.util"
  host_id = "10084"
}

# Lookup by key scoped to a template
data "zabbix_item" "by_template" {
  key         = "system.cpu.util"
  template_id = "10085"
}
