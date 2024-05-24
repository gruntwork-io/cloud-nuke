package telemetry

import (
	"os"

	"github.com/gruntwork-io/go-commons/telemetry"
)

var sendTelemetry = true
var telemetryClient telemetry.MixpanelTelemetryTracker
var cmd = ""
var isCircleCi = false
var account = ""

func InitTelemetry(name string, version string) {
	_, disableTelemetryFlag := os.LookupEnv("DISABLE_TELEMETRY")
	isCircleCi = os.Getenv("CIRCLECI") == "true"
	sendTelemetry = !disableTelemetryFlag
	if sendTelemetry {
		telemetryClient = telemetry.NewMixPanelTelemetryClient("https://t.gruntwork.io/", name, version)
	}
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
}

func SetAccountId(accountId string) {
	account = accountId
}

func TrackEvent(ctx telemetry.EventContext, extraProperties map[string]interface{}) {
	if sendTelemetry {
		ctx.Command = cmd
		extraProperties["isCircleCi"] = isCircleCi
		extraProperties["accountId"] = account
		telemetryClient.TrackEvent(ctx, extraProperties)
	}
}
