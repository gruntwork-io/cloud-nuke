// Package reporting provides event-driven reporting for cloud-nuke operations.
//
// Key components:
//   - Event: Interface for reportable events (ResourceFound, ResourceDeleted, GeneralError)
//   - Collector: Thread-safe event collector that routes events to renderers
//   - Renderer: Interface for output handlers (implemented in renderers package)
//
// The collector is passed explicitly as a function parameter to functions that need it.
package reporting

// Event is the interface for all reportable events.
type Event interface {
	EventType() string
}

// ResourceFound is emitted when a resource is discovered.
type ResourceFound struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	Identifier   string `json:"identifier"`
	Nukable      bool   `json:"nukable"`
	Reason       string `json:"nukable_reason,omitempty"` // empty if nukable
}

func (e ResourceFound) EventType() string { return "resource_found" }

// ResourceDeleted is emitted after a deletion attempt.
type ResourceDeleted struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region,omitempty"`
	Identifier   string `json:"identifier"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"` // empty if success
}

func (e ResourceDeleted) EventType() string { return "resource_deleted" }

// GeneralError is emitted for errors not tied to a specific resource.
type GeneralError struct {
	ResourceType string `json:"resource_type,omitempty"` // optional
	Description  string `json:"description"`
	Error        string `json:"error"`
}

func (e GeneralError) EventType() string { return "general_error" }

// Progress events - these are for live UI updates and are not included in final output.
// Renderers that don't support live progress (like JSON) simply ignore these events.

// ScanProgress is emitted during scanning to show current status.
// CLI renderer uses this to show a spinner with current scan location.
type ScanProgress struct {
	ResourceType string
	Region       string
}

func (e ScanProgress) EventType() string { return "scan_progress" }
