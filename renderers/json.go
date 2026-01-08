package renderers

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/gruntwork-io/cloud-nuke/reporting"
)

// NukeJSONRenderer outputs JSON matching existing cloud-nuke NukeOutput format
type NukeJSONRenderer struct {
	writer    io.Writer
	command   string
	regions   []string
	resources []nukeResourceInfo
	errors    []generalErrorInfo
}

type nukeResourceInfo struct {
	Identifier   string `json:"identifier"`
	ResourceType string `json:"resource_type"`
	Region       string `json:"region,omitempty"`
	Status       string `json:"status"` // "deleted" or "failed"
	Error        string `json:"error,omitempty"`
}

type generalErrorInfo struct {
	ResourceType string `json:"resource_type"`
	Description  string `json:"description"`
	Error        string `json:"error"`
}

// NewNukeJSONRenderer creates a nuke JSON renderer
func NewNukeJSONRenderer(writer io.Writer, command string, regions []string) *NukeJSONRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	if command == "" {
		command = "nuke"
	}
	return &NukeJSONRenderer{
		writer:    writer,
		command:   command,
		regions:   regions,
		resources: make([]nukeResourceInfo, 0),
		errors:    make([]generalErrorInfo, 0),
	}
}

// OnEvent handles events
func (r *NukeJSONRenderer) OnEvent(event reporting.Event) {
	switch e := event.(type) {
	case reporting.ResourceDeleted:
		status := "deleted"
		if !e.Success {
			status = "failed"
		}
		r.resources = append(r.resources, nukeResourceInfo{
			Identifier:   e.Identifier,
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Status:       status,
			Error:        e.Error,
		})
	case reporting.GeneralError:
		r.errors = append(r.errors, generalErrorInfo{
			ResourceType: e.ResourceType,
			Description:  e.Description,
			Error:        e.Error,
		})
	}
}

// Render outputs final JSON (matches existing cloud-nuke NukeOutput format)
func (r *NukeJSONRenderer) Render() error {
	deletedCount := 0
	failedCount := 0
	for _, res := range r.resources {
		if res.Status == "deleted" {
			deletedCount++
		} else {
			failedCount++
		}
	}

	output := NukeOutput{
		Timestamp: time.Now(),
		Command:   r.command,
		Regions:   r.regions,
		Resources: r.resources,
		Errors:    r.errors,
		Summary: NukeSummary{
			Total:         len(r.resources),
			Deleted:       deletedCount,
			Failed:        failedCount,
			GeneralErrors: len(r.errors),
		},
	}

	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// NukeOutput is the complete JSON output for nuke operations
// (matches existing ui/json_types.go)
type NukeOutput struct {
	Timestamp time.Time          `json:"timestamp"`
	Command   string             `json:"command"`
	Account   string             `json:"account,omitempty"`
	Regions   []string           `json:"regions,omitempty"`
	Resources []nukeResourceInfo `json:"resources"`
	Errors    []generalErrorInfo `json:"general_errors,omitempty"`
	Summary   NukeSummary        `json:"summary"`
}

// NukeSummary contains summary counts
type NukeSummary struct {
	Total         int `json:"total"`
	Deleted       int `json:"deleted"`
	Failed        int `json:"failed"`
	GeneralErrors int `json:"general_errors"`
}

// InspectJSONRenderer renders inspection results as JSON
// (matches existing ui/json_types.go InspectOutput format)
type InspectJSONRenderer struct {
	writer    io.Writer
	command   string
	query     QueryParams
	resources []inspectResourceInfo
}

type inspectResourceInfo struct {
	ResourceType  string `json:"resource_type"`
	Region        string `json:"region"`
	Identifier    string `json:"identifier"`
	Nukable       bool   `json:"nukable"`
	NukableReason string `json:"nukable_reason,omitempty"`
}

// QueryParams represents the query parameters used
type QueryParams struct {
	Regions              []string   `json:"regions"`
	ResourceTypes        []string   `json:"resource_types,omitempty"`
	ExcludeAfter         *time.Time `json:"exclude_after,omitempty"`
	IncludeAfter         *time.Time `json:"include_after,omitempty"`
	ListUnaliasedKMSKeys bool       `json:"list_unaliased_kms_keys"`
}

// NewInspectJSONRenderer creates an inspect JSON renderer
func NewInspectJSONRenderer(writer io.Writer, command string, query QueryParams) *InspectJSONRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	if command == "" {
		command = "inspect-aws"
	}
	return &InspectJSONRenderer{
		writer:    writer,
		command:   command,
		query:     query,
		resources: make([]inspectResourceInfo, 0),
	}
}

// OnEvent buffers discovery events
func (r *InspectJSONRenderer) OnEvent(event reporting.Event) {
	if e, ok := event.(reporting.ResourceFound); ok {
		r.resources = append(r.resources, inspectResourceInfo{
			ResourceType:  e.ResourceType,
			Region:        e.Region,
			Identifier:    e.Identifier,
			Nukable:       e.Nukable,
			NukableReason: e.Reason,
		})
	}
}

// Render outputs inspection JSON
func (r *InspectJSONRenderer) Render() error {
	byType := make(map[string]int)
	byRegion := make(map[string]int)
	nukableCount := 0
	nonNukableCount := 0

	for _, res := range r.resources {
		byType[res.ResourceType]++
		byRegion[res.Region]++
		if res.Nukable {
			nukableCount++
		} else {
			nonNukableCount++
		}
	}

	output := InspectOutput{
		Timestamp: time.Now(),
		Command:   r.command,
		Query:     r.query,
		Resources: r.resources,
		Summary: InspectSummary{
			TotalResources: len(r.resources),
			Nukable:        nukableCount,
			NonNukable:     nonNukableCount,
			ByType:         byType,
			ByRegion:       byRegion,
		},
	}

	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// InspectOutput is the complete JSON output for inspect operations
// (matches existing ui/json_types.go)
type InspectOutput struct {
	Timestamp time.Time             `json:"timestamp"`
	Command   string                `json:"command"`
	Query     QueryParams           `json:"query"`
	Resources []inspectResourceInfo `json:"resources"`
	Summary   InspectSummary        `json:"summary"`
}

// InspectSummary contains inspection summary
type InspectSummary struct {
	TotalResources int            `json:"total_resources"`
	Nukable        int            `json:"nukable"`
	NonNukable     int            `json:"non_nukable"`
	ByType         map[string]int `json:"by_type"`
	ByRegion       map[string]int `json:"by_region"`
}
