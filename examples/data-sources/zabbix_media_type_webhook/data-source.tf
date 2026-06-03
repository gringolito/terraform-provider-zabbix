data "zabbix_media_type_webhook" "by_id" {
  id = "10"
}

data "zabbix_media_type_webhook" "by_name" {
  name = "PagerDuty"
}

output "script" {
  value = data.zabbix_media_type_webhook.by_name.script
}
