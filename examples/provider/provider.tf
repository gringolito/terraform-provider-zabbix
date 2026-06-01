# Credentials can be supplied via provider attributes or environment variables:
#   ZABBIX_URL   — base URL of the Zabbix frontend
#   ZABBIX_TOKEN — Zabbix API token
#
# When both are set in the environment the provider block can be empty.
provider "zabbix" {
  zabbix_url = "https://zabbix.example.com"
  api_token  = "your-api-token"
}
