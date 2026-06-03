# Look up a media type by ID
data "zabbix_media_type" "by_id" {
  id = "10"
}

# Look up a media type by name
data "zabbix_media_type" "by_name" {
  name = "Email alerts"
}

output "media_type_id" {
  value = data.zabbix_media_type.by_name.id
}
