resource "terraform_data" "example" {
  input = "fake-string"

  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.zabbix_example.example]
    }
  }
}

action "zabbix_example" "example" {
  config {
    configurable_attribute = "some-value"
  }
}
