package renderers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gruntwork-io/cloud-nuke/reporting"
)

// JSONRenderer outputs results as JSON.
// Output is triggered by terminal events (ScanComplete for inspect, NukeComplete for nuke).
type JSONRenderer struct {
	writer   io.Writer
	command  string
	query    *QueryParams
	regions  []string
	found    []reporting.ResourceFound
	deleted  []reporting.ResourceDeleted
	errors   []reporting.GeneralError
	nukeMode bool // true if NukeStarted was received
}

// NewJSONRenderer creates a JSON renderer.
func NewJSONRenderer(writer io.Writer, cfg JSONRendererConfig) *JSONRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	return &JSONRenderer{
		writer:  writer,
		command: cfg.Command,
		query:   cfg.Query,
		regions: cfg.Regions,
		found:   make([]reporting.ResourceFound, 0),
		deleted: make([]reporting.ResourceDeleted, 0),
		errors:  make([]reporting.GeneralError, 0),
	}
}

// OnEvent collects events and outputs JSON on terminal events.
// Terminal events: ScanComplete (for inspect), NukeComplete (for nuke).
func (r *JSONRenderer) OnEvent(event reporting.Event) {
	switch e := event.(type) {
	case reporting.ResourceFound:
		r.found = append(r.found, e)
	case reporting.ResourceDeleted:
		r.deleted = append(r.deleted, e)
	case reporting.GeneralError:
		r.errors = append(r.errors, e)
	case reporting.NukeStarted:
		r.nukeMode = true
	case reporting.ScanComplete:
		// If not in nuke mode, ScanComplete is terminal - output JSON
		if !r.nukeMode {
			if err := r.renderInspectOutput(); err != nil {
				_, _ = fmt.Fprintf(r.writer, "Error rendering JSON output: %v\n", err)
			}
		}
	case reporting.NukeComplete:
		// NukeComplete is terminal for nuke mode - output JSON
		if err := r.renderNukeOutput(); err != nil {
			_, _ = fmt.Fprintf(r.writer, "Error rendering JSON output: %v\n", err)
		}
	}
}

func (r *JSONRenderer) renderInspectOutput() error {
	byType := make(map[string]int)
	byRegion := make(map[string]int)
	nukableCount := 0
	nonNukableCount := 0

	resources := make([]ResourceInfo, 0, len(r.found))
	for _, e := range r.found {
		resources = append(resources, ResourceInfo{
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Identifier:   e.Identifier,
			Nukable:      e.Nukable,
			Reason:       e.Reason,
		})
		byType[e.ResourceType]++
		byRegion[e.Region]++
		if e.Nukable {
			nukableCount++
		} else {
			nonNukableCount++
		}
	}

	errors := make([]GeneralError, 0, len(r.errors))
	for _, e := range r.errors {
		errors = append(errors, GeneralError{
			ResourceType: e.ResourceType,
			Description:  e.Description,
			Error:        e.Error,
		})
	}

	query := QueryParams{}
	if r.query != nil {
		query = *r.query
	}

	output := InspectOutput{
		Timestamp: time.Now(),
		Command:   r.command,
		Query:     query,
		Resources: resources,
		Errors:    errors,
		Summary: InspectSummary{
			TotalResources: len(r.found),
			Nukable:        nukableCount,
			NonNukable:     nonNukableCount,
			GeneralErrors:  len(r.errors),
			ByType:         byType,
			ByRegion:       byRegion,
		},
	}

	return r.encode(output)
}

func (r *JSONRenderer) renderNukeOutput() error {
	// Build found resources list
	found := make([]ResourceInfo, 0, len(r.found))
	for _, e := range r.found {
		found = append(found, ResourceInfo{
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Identifier:   e.Identifier,
			Nukable:      e.Nukable,
			Reason:       e.Reason,
		})
	}

	// Build deleted resources list
	resources := make([]NukeResourceInfo, 0, len(r.deleted))
	deletedCount := 0
	failedCount := 0

	for _, e := range r.deleted {
		status := "deleted"
		if !e.Success {
			status = "failed"
			failedCount++
		} else {
			deletedCount++
		}
		resources = append(resources, NukeResourceInfo{
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Identifier:   e.Identifier,
			Status:       status,
			Error:        e.Error,
		})
	}

	errors := make([]GeneralError, 0, len(r.errors))
	for _, e := range r.errors {
		errors = append(errors, GeneralError{
			ResourceType: e.ResourceType,
			Description:  e.Description,
			Error:        e.Error,
		})
	}

	output := NukeOutput{
		Timestamp: time.Now(),
		Command:   r.command,
		Regions:   r.regions,
		Found:     found,
		Resources: resources,
		Errors:    errors,
		Summary: NukeSummary{
			Found:         len(r.found),
			Total:         len(r.deleted),
			Deleted:       deletedCount,
			Failed:        failedCount,
			GeneralErrors: len(r.errors),
		},
	}

	return r.encode(output)
}

func (r *JSONRenderer) encode(v any) error {
	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
