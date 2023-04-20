package telemetry

import (
	"github.com/gruntwork-io/go-commons/telemetry"
	"os"
)

var sendTelemetry = true
var telemetryClient telemetry.MixpanelTelemetryTracker
var cmd = ""
var isCircleCi = false
var account = ""

func InitTelemetry(name string, version string, clientId string) {
	_, disableTelemetryFlag := os.LookupEnv("DISABLE_TELEMETRY")
	isCircleCi = os.Getenv("CIRCLECI") == "true"
	clientIdExists := clientId != ""
	sendTelemetry = !disableTelemetryFlag && clientIdExists
	if sendTelemetry {
		cmd = os.Args[1]
		telemetryClient = telemetry.NewMixPanelTelemetryClient(clientId, name, version)
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
