package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

const telemetryURL = "https://t.gruntwork.io/"

// EventContext names the event being reported.
type EventContext struct {
	Command   string
	EventName string
}

// mu guards every variable below. InitTelemetry writes them and TrackEvent
// reads them, and a library caller can do both from different goroutines, so
// the flag that says initialization finished is not enough on its own: without
// a lock the reader can see initialized set before the run ID is published.
var mu sync.RWMutex

var sendTelemetry = true

// initialized records whether InitTelemetry has run. Without it, TrackEvent
// would report events with no app name, version or run ID for anything that
// imports cloud-nuke as a library.
var initialized = false
var appName = ""
var appVersion = ""
var runId = ""
var cmd = ""
var isCircleCi = false
var account = ""

func InitTelemetry(name string, version string) {
	_, disableTelemetryFlag := os.LookupEnv("DISABLE_TELEMETRY")
	enabled := !disableTelemetryFlag
	circleCi := os.Getenv("CIRCLECI") == "true"

	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	mu.Lock()
	defer mu.Unlock()
	sendTelemetry = enabled
	isCircleCi = circleCi
	appName, appVersion, runId = name, version, uuid.New().String()
	cmd = command
	initialized = true
}

func SetAccountId(accountId string) {
	mu.Lock()
	defer mu.Unlock()
	account = accountId
}

func TrackEvent(ctx EventContext, extraProperties map[string]interface{}) {
	mu.RLock()
	enabled, ready := sendTelemetry, initialized
	name, version, id := appName, appVersion, runId
	command, circleCi, accountId := cmd, isCircleCi, account
	mu.RUnlock()

	if !enabled || !ready {
		return
	}

	// The event is sent without the lock held, because it is an HTTP call and
	// the scan reports events from every goroutine at once.
	ctx.Command = command
	extraProperties["isCircleCi"] = circleCi
	extraProperties["accountId"] = accountId

	eventProps := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"context":   name,
		"command":   ctx.Command,
		"version":   version,
	}
	maps.Copy(eventProps, extraProperties)

	body, err := json.Marshal(map[string]interface{}{
		"id":         id,
		"event":      ctx.EventName,
		"eventProps": eventProps,
	})
	if err != nil {
		log.Println(err.Error())
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, telemetryURL, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err.Error())
		return
	}
	if err := resp.Body.Close(); err != nil {
		log.Println(err.Error())
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
	TrackEvent(EventContext{
		EventName: "Start " + commandName,
	}, map[string]interface{}{})

	return func() {
		TrackEvent(EventContext{
			EventName: "End " + commandName,
		}, map[string]interface{}{})
	}
}
