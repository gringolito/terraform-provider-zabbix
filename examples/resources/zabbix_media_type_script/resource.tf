resource "zabbix_media_type_script" "alerts" {
  name      = "Custom script alerts"
  exec_path = "/usr/lib/zabbix/alertscripts/notify.sh"

  exec_params = join("\n", [
    "{ALERT.SENDTO}",
    "{ALERT.SUBJECT}",
    "{ALERT.MESSAGE}",
  ])
}
