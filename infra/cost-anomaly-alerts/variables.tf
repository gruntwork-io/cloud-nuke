variable "slack_webhook_url" {
  description = "Slack incoming webhook URL for cost anomaly alerts"
  type        = string
  sensitive   = true
}

variable "anomaly_threshold_percentage" {
  description = "Percentage threshold for cost anomaly alerts (e.g., 30 = alert when spend is 30% above expected)"
  type        = number
  default     = 30
}

variable "account_name" {
  description = "Account name label used in Slack messages"
  type        = string
  default     = "phxdevops"
}
