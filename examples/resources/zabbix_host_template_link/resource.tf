resource "zabbix_host_group" "linux_servers" {
  name = "Linux servers"
}

resource "zabbix_host" "web_server" {
  host           = "web-server-01"
  host_group_ids = [zabbix_host_group.linux_servers.id]
}

resource "zabbix_template_group" "linux_templates" {
  name = "Linux templates"
}

resource "zabbix_template" "linux_base" {
  host               = "Linux base"
  template_group_ids = [zabbix_template_group.linux_templates.id]
}

resource "zabbix_host_template_link" "web_server_linux_base" {
  host_id     = zabbix_host.web_server.id
  template_id = zabbix_template.linux_base.id
}
