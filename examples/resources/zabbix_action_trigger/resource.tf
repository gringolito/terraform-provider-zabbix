# Minimal trigger action — notify Zabbix administrators on any high-severity event.
resource "zabbix_action_trigger" "notify_admins" {
  name              = "Notify administrators"
  escalation_period = "1h"

  filter {
    evaluation_type = "and_or"

    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "3"
    }
  }

  operations {
    escalation_step_from = 1
    escalation_step_to   = 1
    escalation_period    = "0"

    send_message {
      use_default_message = true
      user_group_ids      = ["7"]
    }
  }

  recovery_operations {
    notify_all_involved = true
  }
}

# Trigger action with custom per-operation subject/message.
resource "zabbix_action_trigger" "custom_message" {
  name              = "Custom message action"
  escalation_period = "30m"

  filter {
    evaluation_type = "and_or"

    condition {
      condition_type = "trigger_severity"
      operator       = "greater_or_equals"
      value          = "4"
    }

    condition {
      condition_type = "maintenance_status"
      operator       = "not_in"
      value          = "1"
    }
  }

  operations {
    escalation_step_from = 1
    escalation_step_to   = 3

    send_message {
      use_default_message = false
      subject             = "ALERT: {EVENT.NAME}"
      message             = "Affected host: {HOST.NAME}"
      user_group_ids      = ["7"]
    }
  }

  recovery_operations {
    send_message {
      use_default_message = false
      subject             = "RESOLVED: {EVENT.NAME}"
      message             = "Problem resolved on {HOST.NAME}"
      user_group_ids      = ["7"]
    }
  }
}
