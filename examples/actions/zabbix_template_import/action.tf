# Standalone invocation — import templates from a local XML export file.
# Run with: terraform apply -invoke=action.zabbix_template_import.example
action "zabbix_template_import" "example" {
  config {
    source = file("templates.xml")
    format = "xml"
  }
}

# Triggered invocation — import templates whenever a template group is created.
resource "zabbix_template_group" "monitoring" {
  name = "Monitoring"

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.zabbix_template_import.on_create]
    }
  }
}

action "zabbix_template_import" "on_create" {
  config {
    source = file("templates.xml")
    format = "xml"
  }
}
