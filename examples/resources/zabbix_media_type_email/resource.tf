resource "zabbix_media_type_email" "alerts" {
  name        = "Email alerts"
  description = "Sends notifications via SMTP."

  smtp_server         = "smtp.example.com"
  smtp_port           = 587
  smtp_email          = "zabbix@example.com"
  smtp_helo           = "example.com"
  smtp_security       = "starttls"
  smtp_authentication = "normal_password"
  username            = "zabbix@example.com"
  password            = var.smtp_password
  content_type        = "html"

  message_templates = [
    {
      eventsource = "trigger"
      recovery    = "operation"
      subject     = "Problem: {EVENT.NAME}"
      message     = "Problem started at {EVENT.TIME} on {EVENT.DATE}\nProblem name: {EVENT.NAME}\nHost: {HOST.NAME}\nSeverity: {EVENT.SEVERITY}"
    },
  ]
}
