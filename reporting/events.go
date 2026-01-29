package reporting

// Event is the interface that all reporting events implement.
type Event interface {
	EventType() string
}

// ScanProgress is emitted during resource discovery to show scanning status.
// Used by CLI renderer to update spinner text.
type ScanProgress struct {
	ResourceType string
	Region       string
}

func (ScanProgress) EventType() string { return "scan_progress" }

// ScanStarted is emitted at the beginning of AWS resource scanning.
// Used by CLI renderer to display query parameters.
// Note: GCP does not emit this event as it has no interesting query parameters to display.
type ScanStarted struct {
	Regions              []string
	ResourceTypes        []string
	ExcludeAfter         string // formatted time string, empty if not set
	IncludeAfter         string // formatted time string, empty if not set
	ListUnaliasedKMSKeys bool
}

func (ScanStarted) EventType() string { return "scan_started" }

// ScanComplete is emitted when all resource scanning is finished.
// Used by CLI renderer to stop spinner and display found resources table.
type ScanComplete struct{}

func (ScanComplete) EventType() string { return "scan_complete" }

// ResourceFound is emitted when a resource is discovered during scanning.
// Used to build the "found resources" table for inspect or pre-nuke display.
type ResourceFound struct {
	ResourceType string
	Region       string
	Identifier   string
	Nukable      bool
	Reason       string // Why not nukable (e.g., "protected by config")
}

func (ResourceFound) EventType() string { return "resource_found" }

// ResourceDeleted is emitted after a deletion attempt.
// Used to build the final nuke results table.
type ResourceDeleted struct {
	ResourceType string
	Region       string
	Identifier   string
	Success      bool
	Error        string // Empty if success
}

func (ResourceDeleted) EventType() string { return "resource_deleted" }

// GeneralError is emitted for non-resource-specific errors during execution.
// Examples: failed to list resources in a region, API errors, etc.
type GeneralError struct {
	ResourceType string
	Description  string
	Error        string
}

func (GeneralError) EventType() string { return "general_error" }

// NukeStarted is emitted at the start of a nuke operation.
// Used by CLI renderer to initialize the progress bar.
type NukeStarted struct {
	Total int // Total resources to nuke
}

func (NukeStarted) EventType() string { return "nuke_started" }

// NukeProgress is emitted when processing a batch of resources.
// Used by CLI renderer to update progress bar title.
type NukeProgress struct {
	ResourceType string
	Region       string
	BatchSize    int
}

func (NukeProgress) EventType() string { return "nuke_progress" }

// NukeComplete is emitted when all nuke operations are finished.
// Used by CLI renderer to stop progress bar and display deletion results.
// Used by JSON renderer to output final JSON.
type NukeComplete struct{}

func (NukeComplete) EventType() string { return "nuke_complete" }
