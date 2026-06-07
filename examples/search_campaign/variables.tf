variable "googleads_developer_token" {
  description = "Google Ads developer token. Prefer TF_VAR_googleads_developer_token or GOOGLEADS_DEVELOPER_TOKEN rather than committing a tfvars file."
  type        = string
  sensitive   = true
}

variable "googleads_customer_id" {
  description = "Google Ads customer ID, with or without dashes."
  type        = string
}

variable "googleads_login_customer_id" {
  description = "Optional manager account customer ID for MCC access. Leave null when not using a manager account."
  type        = string
  default     = null
}
