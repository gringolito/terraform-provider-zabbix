resource "zabbix_media_type_webhook" "pagerduty" {
  name = "PagerDuty"

  max_sessions     = 10
  max_attempts     = 3
  attempt_interval = "10s"

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
