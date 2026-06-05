# Look up by ID
data "zabbix_role" "by_id" {
  id = "5"
}

# Look up by name (works for built-in roles too)
data "zabbix_role" "admin_role" {
  name = "Admin role"
}
