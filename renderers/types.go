package renderers

import (
	"io"
	"os"
	"time"

	"github.com/gruntwork-io/go-commons/errors"
)

// GetOutputWriter returns a writer for the specified output file or stdout if empty.
func GetOutputWriter(outputFile string) (io.Writer, func() error, error) {
	if outputFile == "" {
		return os.Stdout, func() error { return nil }, nil
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, nil, errors.WithStackTrace(err)
	}

	return file, file.Close, nil
}

// JSON Output Types

// InspectOutput represents the JSON output structure for inspect commands.
type InspectOutput struct {
	Timestamp time.Time      `json:"timestamp"`
	Command   string         `json:"command"`
	Query     QueryParams    `json:"query"`
	Resources []ResourceInfo `json:"resources"`
	Errors    []GeneralError `json:"general_errors,omitempty"`
	Summary   InspectSummary `json:"summary"`
}

// QueryParams represents the query parameters used for resource inspection.
type QueryParams struct {
	Regions              []string   `json:"regions"`
	ResourceTypes        []string   `json:"resource_types,omitempty"`
	ExcludeAfter         *time.Time `json:"exclude_after,omitempty"`
	IncludeAfter         *time.Time `json:"include_after,omitempty"`
	ListUnaliasedKMSKeys bool       `json:"list_unaliased_kms_keys"`
}

// ResourceInfo represents information about a single cloud resource.
type ResourceInfo struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	Identifier   string `json:"identifier"`
	Nukable      bool   `json:"nukable"`
	Reason       string `json:"reason,omitempty"`
}

// InspectSummary provides summary statistics for inspection results.
type InspectSummary struct {
	TotalResources int            `json:"total_resources"`
	Nukable        int            `json:"nukable"`
	NonNukable     int            `json:"non_nukable"`
	GeneralErrors  int            `json:"general_errors"`
	ByType         map[string]int `json:"by_type"`
	ByRegion       map[string]int `json:"by_region"`
}

// NukeOutput represents the JSON output structure for nuke commands.
type NukeOutput struct {
	Timestamp time.Time          `json:"timestamp"`
	Command   string             `json:"command"`
	Regions   []string           `json:"regions,omitempty"`
	Found     []ResourceInfo     `json:"found"`
	Resources []NukeResourceInfo `json:"resources"`
	Errors    []GeneralError     `json:"general_errors,omitempty"`
	Summary   NukeSummary        `json:"summary"`
}

// NukeResourceInfo represents information about a resource deletion attempt.
type NukeResourceInfo struct {
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	Identifier   string `json:"identifier"`
	Status       string `json:"status"` // "deleted" or "failed"
	Error        string `json:"error,omitempty"`
}

// GeneralError represents a general error in JSON output.
type GeneralError struct {
	ResourceType string `json:"resource_type"`
	Description  string `json:"description"`
	Error        string `json:"error"`
}

// NukeSummary provides summary statistics for nuke operation results.
type NukeSummary struct {
	Found         int `json:"found"`
	Total         int `json:"total"`
	Deleted       int `json:"deleted"`
	Failed        int `json:"failed"`
	GeneralErrors int `json:"general_errors"`
}

// JSONRendererConfig holds configuration for the JSON renderer.
type JSONRendererConfig struct {
	Command string
	Query   *QueryParams
	Regions []string
}
