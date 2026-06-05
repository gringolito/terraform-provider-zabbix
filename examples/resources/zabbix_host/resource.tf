resource "zabbix_host_group" "linux" {
  name = "Linux servers"
}

resource "zabbix_host" "web_server" {
  host           = "web-srv-01"
  name           = "Web Server 01"
  description    = "Primary web server"
  status         = "enabled"
  host_group_ids = [zabbix_host_group.linux.id]

  tags = [
    { name = "env", value = "production" },
    { name = "role", value = "web" },
  ]

  inventory_mode = "manual"
}
