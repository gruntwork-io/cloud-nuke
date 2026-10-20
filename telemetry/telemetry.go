package telemetry

import (
	"os"

	"github.com/gruntwork-io/go-commons/telemetry"
)

var sendTelemetry = true

// initialized records whether InitTelemetry has run. Without it, TrackEvent
// dereferences a nil HTTP client inside the zero-value tracker, so anything
// importing cloud-nuke as a library crashes on the first event it reports.
var initialized = false
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
	initialized = true
}

func SetAccountId(accountId string) {
	account = accountId
}

func TrackEvent(ctx telemetry.EventContext, extraProperties map[string]interface{}) {
	if sendTelemetry && initialized {
		ctx.Command = cmd
		extraProperties["isCircleCi"] = isCircleCi
		extraProperties["accountId"] = account
		telemetryClient.TrackEvent(ctx, extraProperties)
	}
}

// TrackCommandLifecycle creates a telemetry tracking wrapper for command execution.
// It tracks the start event immediately and returns a cleanup function that tracks the end event.
// Usage:
//
//	func myCommand(c *cli.Context) error {
//	    defer TrackCommandLifecycle("my-command")()
//	    // ... command implementation
//	}
func TrackCommandLifecycle(commandName string) func() {
	TrackEvent(telemetry.EventContext{
		EventName: "Start " + commandName,
	}, map[string]interface{}{})

	return func() {
		TrackEvent(telemetry.EventContext{
			EventName: "End " + commandName,
		}, map[string]interface{}{})
	}
}
