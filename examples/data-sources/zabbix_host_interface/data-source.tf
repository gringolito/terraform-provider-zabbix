# Look up a host interface by its ID
data "zabbix_host_interface" "by_id" {
  id = "42"
}

# Look up a host interface by host ID and type
data "zabbix_host_interface" "agent_iface" {
  host_id = "10084"
  type    = "agent"
}
