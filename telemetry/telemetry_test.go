package telemetry

import (
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
