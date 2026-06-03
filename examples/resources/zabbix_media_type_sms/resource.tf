resource "zabbix_media_type_sms" "alerts" {
  name      = "SMS alerts"
  gsm_modem = "/dev/ttyS0"
}
