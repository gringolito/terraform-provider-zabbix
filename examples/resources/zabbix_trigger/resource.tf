# Basic trigger with expression recovery mode (default)
resource "zabbix_trigger" "high_cpu" {
  description = "High CPU utilization"
  expression  = "last(/web-srv-01/system.cpu.util)>90"
  priority    = "high"
}

# Trigger with recovery expression
resource "zabbix_trigger" "disk_low" {
  description         = "Disk space is low"
  expression          = "last(/web-srv-01/vfs.fs.size[/,pfree])<10"
  priority            = "average"
  recovery_mode       = "recovery_expression"
  recovery_expression = "last(/web-srv-01/vfs.fs.size[/,pfree])>20"
  comments            = "Alert when disk free space drops below 10%, recover when above 20%"
}

# Trigger with no recovery (manual close only)
resource "zabbix_trigger" "service_down" {
  description   = "Service is unavailable"
  expression    = "last(/web-srv-01/net.tcp.service[http,,80])=0"
  priority      = "disaster"
  recovery_mode = "none"
  manual_close  = true
}

# Trigger with tags
resource "zabbix_trigger" "high_load" {
  description = "High load average"
  expression  = "avg(/web-srv-01/system.cpu.load[all,avg1],5m)>4"
  priority    = "warning"

  tags = [
    { name = "team", value = "ops" },
    { name = "environment", value = "production" },
  ]
}
