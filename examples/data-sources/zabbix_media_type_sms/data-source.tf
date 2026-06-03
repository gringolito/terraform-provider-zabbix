data "zabbix_media_type_sms" "by_id" {
  id = "10"
}

data "zabbix_media_type_sms" "by_name" {
  name = "SMS alerts"
}

output "gsm_modem" {
  value = data.zabbix_media_type_sms.by_name.gsm_modem
}
