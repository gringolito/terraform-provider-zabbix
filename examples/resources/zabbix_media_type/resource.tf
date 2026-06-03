# Email media type
resource "zabbix_media_type" "email" {
  name        = "Email alerts"
  type        = "email"
  description = "Sends notifications via SMTP."

  email_settings = {
    smtp_server         = "smtp.example.com"
    smtp_port           = 587
    smtp_email          = "zabbix@example.com"
    smtp_helo           = "example.com"
    smtp_security       = "starttls"
    smtp_authentication = "normal_password"
    username            = "zabbix@example.com"
    password            = var.smtp_password
    content_type        = "html"
  }
}

# Webhook media type
resource "zabbix_media_type" "webhook" {
  name = "PagerDuty webhook"
  type = "webhook"

  max_sessions     = 10
  max_attempts     = 3
  attempt_interval = "10s"

  webhook_settings = {
    script  = file("${path.module}/webhook.js")
    timeout = "30s"

    parameters = [
      {
        name  = "URL"
        value = "https://events.pagerduty.com/v2/enqueue"
      },
      {
        name  = "Token"
        value = var.pagerduty_token
      },
    ]
  }
}
