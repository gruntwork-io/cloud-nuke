package telemetry

import (
	"os"

	"github.com/gruntwork-io/go-commons/telemetry"
)

// Telemetry Event Names
// These constants define the event names used for tracking usage and errors
const (
	EventInitialized          = "initialized"
	EventStartAWS             = "Start aws"
	EventEndAWS               = "End aws"
	EventStartAWSDefaults     = "Start aws-defaults"
	EventEndAWSDefaults       = "End aws-defaults"
	EventStartAWSInspect      = "Start aws-inspect"
	EventEndAWSInspect        = "End aws-inspect"
	EventStartGCP             = "Start gcp"
	EventEndGCP               = "End gcp"
	EventStartGCPInspect      = "Start gcp-inspect"
	EventEndGCPInspect        = "End gcp-inspect"
	EventReadingConfig        = "Reading config file"
	EventErrorReadingConfig   = "Error reading config file"
	EventNoResources          = "No resources to nuke"
	EventSkippingDryRun       = "Skipping nuke, dryrun set"
	EventAwaitingConfirmation = "Awaiting nuke confirmation"
	EventErrorConfirming      = "Error confirming nuke"
	EventUserAborted          = "User aborted nuke"
	EventForcingNuke          = "Forcing nuke in 10 seconds"
	EventErrorGettingResources = "Error getting resources"
	EventErrorInspecting      = "Error inspecting resources"
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
