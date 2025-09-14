package ui

import "time"

// InspectOutput represents the JSON output structure for inspect commands
type InspectOutput struct {
	Timestamp time.Time      `json:"timestamp"`
	Command   string         `json:"command"`
	Query     QueryParams    `json:"query"`
	Resources []ResourceInfo `json:"resources"`
	Summary   InspectSummary `json:"summary"`
}

// QueryParams represents the query parameters used for resource inspection
type QueryParams struct {
	Regions              []string   `json:"regions"`
	ResourceTypes        []string   `json:"resource_types,omitempty"`
	ExcludeAfter         *time.Time `json:"exclude_after,omitempty"`
	IncludeAfter         *time.Time `json:"include_after,omitempty"`
	ListUnaliasedKMSKeys bool       `json:"list_unaliased_kms_keys"`
}

// ResourceInfo represents information about a single cloud resource
type ResourceInfo struct {
	ResourceType  string            `json:"resource_type"`
	Region        string            `json:"region"`
	Identifier    string            `json:"identifier"`
	Nukable       bool              `json:"nukable"`
	NukableReason string            `json:"nukable_reason,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	CreatedTime   *time.Time        `json:"created_time,omitempty"`
}

// InspectSummary provides summary statistics for inspection results
type InspectSummary struct {
	TotalResources int            `json:"total_resources"`
	Nukable        int            `json:"nukable"`
	NonNukable     int            `json:"non_nukable"`
	ByType         map[string]int `json:"by_type"`
	ByRegion       map[string]int `json:"by_region"`
}

// NukeOutput represents the JSON output structure for nuke commands
type NukeOutput struct {
	Timestamp time.Time          `json:"timestamp"`
	Command   string             `json:"command"`
	Account   string             `json:"account,omitempty"`
	Regions   []string           `json:"regions,omitempty"`
	Resources []NukeResourceInfo `json:"resources"`
	Errors    []GeneralErrorInfo `json:"general_errors,omitempty"`
	Summary   NukeSummary        `json:"summary"`
}

// NukeResourceInfo represents information about a resource deletion attempt
type NukeResourceInfo struct {
	Identifier   string `json:"identifier"`
	ResourceType string `json:"resource_type"`
	Status       string `json:"status"` // "deleted" or "failed"
	Error        string `json:"error,omitempty"`
}

// GeneralErrorInfo represents a general error that occurred during execution
type GeneralErrorInfo struct {
	ResourceType string `json:"resource_type"`
	Description  string `json:"description"`
	Error        string `json:"error"`
}

// NukeSummary provides summary statistics for nuke operation results
type NukeSummary struct {
	Total         int `json:"total"`
	Deleted       int `json:"deleted"`
	Failed        int `json:"failed"`
	GeneralErrors int `json:"general_errors"`
}