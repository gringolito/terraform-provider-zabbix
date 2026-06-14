data "zabbix_authentication" "current" {}

output "auth_type" {
  value = data.zabbix_authentication.current.authentication_type
}
