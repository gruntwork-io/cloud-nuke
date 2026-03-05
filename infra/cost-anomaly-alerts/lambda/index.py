import json
import logging
import os
import urllib.request
from urllib.parse import urlparse

logger = logging.getLogger()
logger.setLevel(os.environ.get("LOG_LEVEL", "INFO"))


def _fmt_dollars(val):
    try:
        return f"${float(val):.2f}"
    except (TypeError, ValueError):
        return "N/A"


def handler(event, context):
    webhook_url = os.environ.get("SLACK_WEBHOOK_URL", "")
    account_name = os.environ.get("ACCOUNT_NAME", "AWS")

    if not webhook_url:
        logger.error("SLACK_WEBHOOK_URL environment variable is not set")
        return {"statusCode": 500, "body": "Missing configuration"}

    parsed = urlparse(webhook_url)
    if parsed.hostname != "hooks.slack.com":
        logger.error("SLACK_WEBHOOK_URL does not point to hooks.slack.com, aborting")
        return {"statusCode": 400, "body": "Invalid webhook URL"}

    records = event.get("Records", [])
    if not records:
        logger.warning("Received event with no Records")
        return {"statusCode": 200, "body": "No records"}

    for record in records:
        try:
            message = json.loads(record["Sns"]["Message"])
        except (KeyError, json.JSONDecodeError):
            logger.exception("Failed to parse SNS message")
            continue

        # Each Cost Anomaly Detection notification contains a single anomaly
        # as the top-level object (no wrapping array).
        root_causes = message.get("rootCauses", [])
        impact = message.get("impact", {})
        total_impact = impact.get("totalImpact", 0)

        service = root_causes[0].get("service", "Unknown") if root_causes else "Unknown"
        region = root_causes[0].get("region", "Unknown") if root_causes else "Unknown"
        account = root_causes[0].get("linkedAccount", "Unknown") if root_causes else "Unknown"

        if len(root_causes) > 1:
            service += f" (+{len(root_causes) - 1} more)"

        expected = impact.get("totalExpectedSpend", None)
        actual = impact.get("totalActualSpend", None)
        anomaly_link = message.get("anomalyDetailsLink", "")

        logger.info(
            "Anomaly detected: service=%s region=%s impact=%s",
            service, region, total_impact,
        )

        fields = [
            {"type": "mrkdwn", "text": f"*Service:*\n{service}"},
            {"type": "mrkdwn", "text": f"*Region:*\n{region}"},
            {"type": "mrkdwn", "text": f"*Account:*\n{account}"},
            {"type": "mrkdwn", "text": f"*Impact:*\n{_fmt_dollars(total_impact)}"},
            {"type": "mrkdwn", "text": f"*Expected Spend:*\n{_fmt_dollars(expected)}"},
            {"type": "mrkdwn", "text": f"*Actual Spend:*\n{_fmt_dollars(actual)}"},
        ]

        blocks = [
            {
                "type": "header",
                "text": {
                    "type": "plain_text",
                    "text": f"Cost Anomaly Detected — {account_name}",
                },
            },
            {"type": "section", "fields": fields},
        ]

        if anomaly_link:
            blocks.append({
                "type": "actions",
                "elements": [{
                    "type": "button",
                    "text": {"type": "plain_text", "text": "View in AWS Console"},
                    "url": anomaly_link,
                }],
            })

        slack_message = {
            "text": f":money_with_wings: *Cost Anomaly Detected — {account_name}*",
            "blocks": blocks,
        }

        req = urllib.request.Request(
            webhook_url,
            data=json.dumps(slack_message).encode("utf-8"),
            headers={"Content-Type": "application/json"},
        )
        try:
            resp = urllib.request.urlopen(req, timeout=10)
            body = resp.read().decode("utf-8")
            if body != "ok":
                logger.warning("Slack returned non-ok response: %s", body)
        except Exception:
            logger.exception("Failed to post to Slack")

    return {"statusCode": 200, "body": "OK"}
