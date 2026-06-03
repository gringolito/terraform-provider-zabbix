data "zabbix_media_type_script" "by_id" {
  id = "10"
}

data "zabbix_media_type_script" "by_name" {
  name = "Custom script alerts"
}

output "exec_path" {
  value = data.zabbix_media_type_script.by_name.exec_path
}
