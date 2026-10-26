package telemetry

import (
	"sync"
	"testing"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

// TestTrackEventBeforeInitDoesNothing pins that a caller who never initialized
// telemetry does not crash on the first event cloud-nuke reports. sendTelemetry
// defaults to true while telemetryClient is a zero value whose HTTP client is
// nil, so TrackEvent used to dereference it. The CLI always initializes first,
// which is why this only ever reached a library caller: aws.GetAllResources
// reports an event on its success path.
func TestTrackEventBeforeInitDoesNothing(t *testing.T) {
	TrackEvent(commonTelemetry.EventContext{EventName: "probe"}, map[string]interface{}{})
}

// TestInitAndTrackConcurrently pins that initializing telemetry while events
// are reported is not a data race. The CLI initializes before it scans, so this
// only reaches a library caller, which is the same caller the nil client
// reached. DISABLE_TELEMETRY keeps the test off the network: TrackEvent still
// reads every variable InitTelemetry writes, which is what `go test -race`
// watches.
func TestInitAndTrackConcurrently(t *testing.T) {
	t.Setenv("DISABLE_TELEMETRY", "1")
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		sendTelemetry, initialized, cmd, isCircleCi, account = true, false, "", false, ""
		telemetryClient = commonTelemetry.MixpanelTelemetryTracker{}
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); InitTelemetry("cloud-nuke", "test") }()
		go func() { defer wg.Done(); SetAccountId("account") }()
		go func() {
			defer wg.Done()
			TrackEvent(commonTelemetry.EventContext{EventName: "probe"}, map[string]interface{}{})
		}()
	}
	wg.Wait()
}
