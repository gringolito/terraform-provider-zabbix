resource "zabbix_template_group" "linux_templates" {
  name = "Linux templates"
}

resource "zabbix_template" "linux_base" {
  host               = "Linux base"
  template_group_ids = [zabbix_template_group.linux_templates.id]
}

resource "zabbix_template" "linux_extended" {
  host               = "Linux extended"
  template_group_ids = [zabbix_template_group.linux_templates.id]
}

resource "zabbix_template_link" "extended_inherits_base" {
  template_id        = zabbix_template.linux_extended.id
  linked_template_id = zabbix_template.linux_base.id
}
