# Look up by ID
data "zabbix_template_group" "by_id" {
  id = "2"
}

# Look up by name
data "zabbix_template_group" "by_name" {
  name = "Linux templates"
}
