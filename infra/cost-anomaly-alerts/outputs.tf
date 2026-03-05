output "sns_topic_arn" {
  description = "ARN of the cost anomaly alerts SNS topic"
  value       = aws_sns_topic.cost_anomaly_alerts.arn
}

output "lambda_function_arn" {
  description = "ARN of the Slack forwarder Lambda function"
  value       = aws_lambda_function.cost_anomaly_to_slack.arn
}

output "anomaly_monitor_arn" {
  description = "ARN of the Cost Anomaly Detection monitor"
  value       = aws_ce_anomaly_monitor.service_monitor.arn
}

output "anomaly_subscription_arn" {
  description = "ARN of the Cost Anomaly Detection subscription"
  value       = aws_ce_anomaly_subscription.cost_alerts.arn
}
