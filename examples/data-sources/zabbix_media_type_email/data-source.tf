data "zabbix_media_type_email" "by_id" {
  id = "10"
}

data "zabbix_media_type_email" "by_name" {
  name = "Email alerts"
}

output "smtp_server" {
  value = data.zabbix_media_type_email.by_name.smtp_server
}
