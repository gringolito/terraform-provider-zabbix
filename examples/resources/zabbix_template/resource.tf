resource "zabbix_template_group" "linux_templates" {
  name = "Linux templates"
}

resource "zabbix_template" "linux_base" {
  host        = "Linux base"
  name        = "Linux base template"
  description = "Base monitoring for Linux hosts"

  template_group_ids = [zabbix_template_group.linux_templates.id]

  macros = {
    "{$AGENT.PORT}"  = "10050"
    "{$AGENT.TIMEOUT}" = "3"
  }
}

resource "zabbix_template" "linux_extended" {
  host = "Linux extended"

  template_group_ids  = [zabbix_template_group.linux_templates.id]
  linked_template_ids = [zabbix_template.linux_base.id]
}
