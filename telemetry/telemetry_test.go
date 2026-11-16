package telemetry

import (
	"sync"
	"testing"
)

// TestTrackEventBeforeInitDoesNothing pins that a caller who never initialized
// telemetry does not crash or send anything on the first event cloud-nuke
// reports, even though sendTelemetry defaults to true. The CLI always
// initializes first, which is why this only ever reached a library caller:
// aws.GetAllResources reports an event on its success path.
func TestTrackEventBeforeInitDoesNothing(t *testing.T) {
	TrackEvent(EventContext{EventName: "probe"}, map[string]interface{}{})
}

// TestInitAndTrackConcurrently pins that initializing telemetry while events
// are reported is not a data race. The CLI initializes before it scans, so this
// only reaches a library caller. DISABLE_TELEMETRY keeps the test off the
// network: TrackEvent still reads every variable InitTelemetry writes, which is
// what `go test -race` watches.
func TestInitAndTrackConcurrently(t *testing.T) {
	t.Setenv("DISABLE_TELEMETRY", "1")
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		sendTelemetry, initialized, cmd, isCircleCi, account = true, false, "", false, ""
		appName, appVersion, runId = "", "", ""
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); InitTelemetry("cloud-nuke", "test") }()
		go func() { defer wg.Done(); SetAccountId("account") }()
		go func() {
			defer wg.Done()
			TrackEvent(EventContext{EventName: "probe"}, map[string]interface{}{})
		}()
	}
	wg.Wait()
}
