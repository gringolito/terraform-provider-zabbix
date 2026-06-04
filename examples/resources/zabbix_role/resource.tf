# Minimal role (no explicit rules — Zabbix fills defaults)
resource "zabbix_role" "viewer" {
  name = "Read-only viewer"
  type = "user"
}

# Role with explicit API restrictions
resource "zabbix_role" "api_limited" {
  name = "API limited admin"
  type = "admin"

  rules {
    api_access = true
    api_mode   = "allow"
    api_methods = [
      "host.get",
      "item.get",
      "trigger.get",
    ]

    ui_default_access = true
  }
}
