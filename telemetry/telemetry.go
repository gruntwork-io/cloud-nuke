package telemetry

import (
	"os"
	"sync"

	"github.com/gruntwork-io/go-commons/telemetry"
)

// mu guards every variable below. InitTelemetry writes them and TrackEvent
// reads them, and a library caller can do both from different goroutines, so
// the flag that says initialization finished is not enough on its own: without
// a lock the reader can see initialized set before telemetryClient is
// published, which is the nil dereference again by another route.
var mu sync.RWMutex

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
	enabled := !disableTelemetryFlag
	circleCi := os.Getenv("CIRCLECI") == "true"

	var client telemetry.MixpanelTelemetryTracker
	if enabled {
		client = telemetry.NewMixPanelTelemetryClient("https://t.gruntwork.io/", name, version)
	}

	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	mu.Lock()
	defer mu.Unlock()
	sendTelemetry = enabled
	isCircleCi = circleCi
	telemetryClient = client
	cmd = command
	initialized = true
}

func SetAccountId(accountId string) {
	mu.Lock()
	defer mu.Unlock()
	account = accountId
}

func TrackEvent(ctx telemetry.EventContext, extraProperties map[string]interface{}) {
	mu.RLock()
	enabled, ready := sendTelemetry, initialized
	client, command, circleCi, accountId := telemetryClient, cmd, isCircleCi, account
	mu.RUnlock()

	if !enabled || !ready {
		return
	}

	// The event is sent without the lock held, because the tracker makes an
	// HTTP call and the scan reports events from every goroutine at once.
	ctx.Command = command
	extraProperties["isCircleCi"] = circleCi
	extraProperties["accountId"] = accountId
	client.TrackEvent(ctx, extraProperties)
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
