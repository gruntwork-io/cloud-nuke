package renderers

import (
	"time"

	"github.com/gruntwork-io/cloud-nuke/reporting"
)

// QueryParams represents query parameters for inspect operations
type QueryParams struct {
	Regions              []string   `json:"regions"`
	ResourceTypes        []string   `json:"resource_types,omitempty"`
	ExcludeAfter         *time.Time `json:"exclude_after,omitempty"`
	IncludeAfter         *time.Time `json:"include_after,omitempty"`
	ListUnaliasedKMSKeys bool       `json:"list_unaliased_kms_keys"`
}

// JSONRendererConfig holds configuration for creating a JSON renderer
type JSONRendererConfig struct {
	Command string
	Query   *QueryParams // for inspect
	Regions []string     // for nuke
}

// InspectOutput is the JSON output for inspect operations
type InspectOutput struct {
	Timestamp time.Time                 `json:"timestamp"`
	Command   string                    `json:"command"`
	Query     QueryParams               `json:"query"`
	Resources []reporting.ResourceFound `json:"resources"`
	Errors    []reporting.GeneralError  `json:"general_errors,omitempty"`
	Summary   InspectSummary            `json:"summary"`
}

// InspectSummary contains inspection summary
type InspectSummary struct {
	TotalResources int            `json:"total_resources"`
	Nukable        int            `json:"nukable"`
	NonNukable     int            `json:"non_nukable"`
	GeneralErrors  int            `json:"general_errors"`
	ByType         map[string]int `json:"by_type"`
	ByRegion       map[string]int `json:"by_region"`
}

// NukeOutput is the JSON output for nuke operations
type NukeOutput struct {
	Timestamp time.Time                   `json:"timestamp"`
	Command   string                      `json:"command"`
	Account   string                      `json:"account,omitempty"`
	Regions   []string                    `json:"regions,omitempty"`
	Resources []reporting.ResourceDeleted `json:"resources"`
	Errors    []reporting.GeneralError    `json:"general_errors,omitempty"`
	Summary   NukeSummary                 `json:"summary"`
}

// NukeSummary contains nuke summary counts
type NukeSummary struct {
	Total         int `json:"total"`
	Deleted       int `json:"deleted"`
	Failed        int `json:"failed"`
	GeneralErrors int `json:"general_errors"`
}
