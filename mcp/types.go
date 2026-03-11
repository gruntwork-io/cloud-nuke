package mcp

// ServerConfig holds configuration for the MCP server, populated from CLI flags.
type ServerConfig struct {
	ReadOnly             bool
	AllowedRegions       []string
	AllowedResourceTypes []string
	AllowedProjects      []string
	MaxResourcesPerNuke  int
}

// DefaultServerConfig returns a ServerConfig with safe defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		MaxResourcesPerNuke: DefaultMaxResourcesPerNuke,
	}
}

// InspectResult is the JSON response for inspect_resources.
type InspectResult struct {
	Resources []ResourceInfo     `json:"resources"`
	Errors    []GeneralErrorInfo `json:"errors,omitempty"`
	Summary   InspectSummary     `json:"summary"`
}

// NukeResult is the JSON response for nuke_resources.
type NukeResult struct {
	DryRun    bool               `json:"dry_run"`
	Resources []ResourceInfo     `json:"resources,omitempty"`
	Deleted   []DeletedInfo      `json:"deleted,omitempty"`
	Errors    []GeneralErrorInfo `json:"errors,omitempty"`
	Summary   NukeSummary        `json:"summary"`
}

// ResourceInfo represents a discovered resource.
type ResourceInfo struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	Identifier   string `json:"identifier"`
	Nukable      bool   `json:"nukable"`
	Reason       string `json:"reason,omitempty"`
}

// DeletedInfo represents a deletion attempt result.
type DeletedInfo struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	Identifier   string `json:"identifier"`
	Status       string `json:"status"` // "deleted" or "failed"
	Error        string `json:"error,omitempty"`
}

// GeneralErrorInfo represents a non-resource-specific error.
type GeneralErrorInfo struct {
	ResourceType string `json:"resource_type,omitempty"`
	Description  string `json:"description"`
	Error        string `json:"error"`
}

// InspectSummary provides summary counts for inspect results.
type InspectSummary struct {
	TotalResources int            `json:"total_resources"`
	Nukable        int            `json:"nukable"`
	NonNukable     int            `json:"non_nukable"`
	GeneralErrors  int            `json:"general_errors"`
	ByType         map[string]int `json:"by_type"`
	ByRegion       map[string]int `json:"by_region"`
}

// NukeSummary provides summary counts for nuke results.
type NukeSummary struct {
	Found         int `json:"found"`
	Deleted       int `json:"deleted"`
	Failed        int `json:"failed"`
	GeneralErrors int `json:"general_errors"`
}

// ValidateConfigResult is the JSON response for validate_config.
type ValidateConfigResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}
