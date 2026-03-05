terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

data "aws_caller_identity" "current" {}

# -----------------------------------------------------------------------------
# SNS Topic
# -----------------------------------------------------------------------------

resource "aws_sns_topic" "cost_anomaly_alerts" {
  name = "cost-anomaly-alerts"
}

resource "aws_sns_topic_policy" "cost_anomaly_alerts" {
  arn = aws_sns_topic.cost_anomaly_alerts.arn

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCostAnomalyDetection"
        Effect    = "Allow"
        Principal = { Service = "costalerts.amazonaws.com" }
        Action    = "SNS:Publish"
        Resource  = aws_sns_topic.cost_anomaly_alerts.arn
        Condition = {
          StringEquals = {
            "aws:SourceAccount" = data.aws_caller_identity.current.account_id
          }
        }
      }
    ]
  })
}

# -----------------------------------------------------------------------------
# Lambda IAM Role
# -----------------------------------------------------------------------------

resource "aws_iam_role" "lambda_exec" {
  name = "cost-anomaly-to-slack-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action    = "sts:AssumeRole"
        Effect    = "Allow"
        Principal = { Service = "lambda.amazonaws.com" }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# -----------------------------------------------------------------------------
# Lambda Function
# -----------------------------------------------------------------------------

data "archive_file" "lambda_zip" {
  type        = "zip"
  source_file = "${path.module}/lambda/index.py"
  output_path = "${path.module}/lambda/function.zip"
}

resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/cost-anomaly-to-slack"
  retention_in_days = 30
}

resource "aws_lambda_function" "cost_anomaly_to_slack" {
  function_name                  = "cost-anomaly-to-slack"
  role                           = aws_iam_role.lambda_exec.arn
  handler                        = "index.handler"
  runtime                        = "python3.12"
  timeout                        = 30
  reserved_concurrent_executions = 5
  filename                       = data.archive_file.lambda_zip.output_path
  source_code_hash               = data.archive_file.lambda_zip.output_base64sha256

  environment {
    variables = {
      SLACK_WEBHOOK_URL = var.slack_webhook_url
      ACCOUNT_NAME      = var.account_name
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda_logs]
}

# -----------------------------------------------------------------------------
# SNS → Lambda Subscription
# -----------------------------------------------------------------------------

resource "aws_sns_topic_subscription" "lambda" {
  topic_arn = aws_sns_topic.cost_anomaly_alerts.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.cost_anomaly_to_slack.arn
}

resource "aws_lambda_permission" "sns_invoke" {
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cost_anomaly_to_slack.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.cost_anomaly_alerts.arn
}

# -----------------------------------------------------------------------------
# Cost Anomaly Detection
# -----------------------------------------------------------------------------

resource "aws_ce_anomaly_monitor" "service_monitor" {
  name              = "cost-anomaly-service-monitor"
  monitor_type      = "DIMENSIONAL"
  monitor_dimension = "SERVICE"
}

resource "aws_ce_anomaly_subscription" "cost_alerts" {
  name = "cost-anomaly-alerts"

  monitor_arn_list = [aws_ce_anomaly_monitor.service_monitor.arn]

  frequency = "IMMEDIATE"

  threshold_expression {
    dimension {
      key           = "ANOMALY_TOTAL_IMPACT_PERCENTAGE"
      match_options = ["GREATER_THAN_OR_EQUAL"]
      values        = [tostring(var.anomaly_threshold_percentage)]
    }
  }

  subscriber {
    type    = "SNS"
    address = aws_sns_topic.cost_anomaly_alerts.arn
  }
}
