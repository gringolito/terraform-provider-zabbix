resource "zabbix_authentication" "example" {
  authentication_type = "internal"
  passwd_min_length   = 12
  passwd_check_rules  = ["case_sensitive_letters", "digits", "special_characters", "avoid_common_passwords"]
  mfa_status          = "disabled"
}
