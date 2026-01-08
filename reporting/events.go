// Package reporting provides event-driven reporting for cloud-nuke operations.
//
// This package implements a Collector/Renderer pattern that replaces the deprecated
// global-state based reporting in the report package. Key components:
//
//   - Event: Interface for all reportable events (ResourceFound, ResourceDeleted, GeneralError)
//   - Collector: Thread-safe event collector that routes events to renderers
//   - Renderer: Interface for output handlers (implemented in the renderers package)
//
// Usage:
//
//	collector := reporting.NewCollector()
//	collector.AddRenderer(renderers.NewNukeCLIRenderer(os.Stdout))
//	ctx := reporting.WithCollector(context.Background(), collector)
//
//	// Events are emitted during operations via:
//	// - collector.RecordFound() for discovered resources
//	// - collector.RecordDeleted() for deletion results
//	// - collector.RecordError() for general errors
//
//	// Render final output:
//	collector.Complete()
package reporting

// Event represents something that happened during a nuke/inspect operation.
// Events are emitted in real-time and can be consumed by renderers.
type Event interface {
	EventType() string
}

// ResourceFound is emitted when a resource is discovered during inspection.
type ResourceFound struct {
	ResourceType string
	Region       string
	Identifier   string
	Nukable      bool
	Reason       string // why not nukable (empty if nukable)
}

func (e ResourceFound) EventType() string { return "resource_found" }

// ResourceDeleted is emitted when a deletion attempt completes.
type ResourceDeleted struct {
	ResourceType string
	Region       string
	Identifier   string
	Success      bool
	Error        string // empty if success
}

func (e ResourceDeleted) EventType() string { return "resource_deleted" }

// GeneralError is emitted for errors not tied to a specific resource.
type GeneralError struct {
	ResourceType string // optional, may be empty
	Description  string
	Error        string
}

func (e GeneralError) EventType() string { return "general_error" }
