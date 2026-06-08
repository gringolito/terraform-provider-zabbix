resource "zabbix_host_group" "linux" {
  name = "Linux servers"
}

resource "zabbix_host" "web_server" {
  host           = "web-srv-01"
  host_group_ids = [zabbix_host_group.linux.id]
}

# Agent interface (default)
resource "zabbix_host_interface" "agent" {
  host_id = zabbix_host.web_server.id
  type    = "agent"
  use_ip  = true
  ip      = "192.168.1.10"
  dns     = ""
  port    = "10050"
  main    = true
}

# SNMPv2c interface
resource "zabbix_host_interface" "snmp_v2c" {
  host_id = zabbix_host.web_server.id
  type    = "snmp"
  use_ip  = true
  ip      = "192.168.1.10"
  dns     = ""
  port    = "161"
  main    = true

  snmp = {
    version   = "v2c"
    community = "public"
  }
}

# SNMPv3 interface
resource "zabbix_host_interface" "snmp_v3" {
  host_id = zabbix_host.web_server.id
  type    = "snmp"
  use_ip  = true
  ip      = "192.168.1.10"
  dns     = ""
  port    = "161"
  main    = false

  snmp = {
    version         = "v3"
    security_name   = "admin"
    security_level  = "authPriv"
    auth_protocol   = "sha256"
    auth_passphrase = "auth-secret"
    priv_protocol   = "aes128"
    priv_passphrase = "priv-secret"
  }
}

# IPMI interface
resource "zabbix_host_interface" "ipmi" {
  host_id = zabbix_host.web_server.id
  type    = "ipmi"
  use_ip  = true
  ip      = "192.168.1.10"
  dns     = ""
  port    = "623"
  main    = true
}

# JMX interface
resource "zabbix_host_interface" "jmx" {
  host_id = zabbix_host.web_server.id
  type    = "jmx"
  use_ip  = true
  ip      = "192.168.1.10"
  dns     = ""
  port    = "12345"
  main    = true
}
