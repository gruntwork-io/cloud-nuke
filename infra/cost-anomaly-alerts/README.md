# Cost Anomaly Alerts

AWS Cost Anomaly Detection → Slack alerting for the phxdevops account (087285199408).

## Architecture

```
┌─────────────────────────┐
│  Cost Anomaly Detection │
│  (ML per-service model) │
└───────────┬─────────────┘
            │ anomaly detected (≥30% above baseline)
            ▼
┌─────────────────────────┐
│  SNS Topic              │
│  cost-anomaly-alerts    │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Lambda                 │
│  cost-anomaly-to-slack  │
│  (Python 3.12)          │
└───────────┬─────────────┘
            │ POST
            ▼
┌─────────────────────────┐
│  Slack Webhook          │
│  #cost-alerts channel   │
└─────────────────────────┘
```

## Setup

```bash
cd infra/cost-anomaly-alerts

# Create terraform.tfvars (gitignored)
cat > terraform.tfvars <<EOF
slack_webhook_url = "https://hooks.slack.com/services/..."
EOF

terraform init
terraform plan
terraform apply
```

## Variables

| Name | Description | Default |
|------|-------------|---------|
| `slack_webhook_url` | Slack incoming webhook URL | (required, sensitive) |
| `anomaly_threshold_percentage` | % above expected spend to trigger alert | `30` |
| `account_name` | Label in Slack messages | `phxdevops` |

## Cloud-Nuke Protection

The SNS topic, Lambda function, and IAM role are excluded in `.github/nuke_config.yml`.

## Cost

$0/mo — all resources within free tier.
