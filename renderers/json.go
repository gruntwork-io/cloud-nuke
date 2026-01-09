package renderers

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/gruntwork-io/cloud-nuke/reporting"
)

// JSONRenderer outputs results as JSON.
// Auto-detects output format based on collected events.
type JSONRenderer struct {
	writer  io.Writer
	command string
	query   *QueryParams // for inspect operations
	regions []string     // for nuke operations
	found   []reporting.ResourceFound
	deleted []reporting.ResourceDeleted
	errors  []reporting.GeneralError
}

// NewJSONRenderer creates a JSON renderer
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

// OnEvent collects events
func (r *JSONRenderer) OnEvent(event reporting.Event) {
	switch e := event.(type) {
	case reporting.ResourceFound:
		r.found = append(r.found, e)
	case reporting.ResourceDeleted:
		r.deleted = append(r.deleted, e)
	case reporting.GeneralError:
		r.errors = append(r.errors, e)
	}
}

// Render outputs JSON - auto-detects format based on collected events
func (r *JSONRenderer) Render() error {
	if len(r.deleted) > 0 {
		return r.renderNukeOutput()
	}
	return r.renderInspectOutput()
}

func (r *JSONRenderer) renderInspectOutput() error {
	byType := make(map[string]int)
	byRegion := make(map[string]int)
	nukableCount := 0
	nonNukableCount := 0

	for _, e := range r.found {
		byType[e.ResourceType]++
		byRegion[e.Region]++
		if e.Nukable {
			nukableCount++
		} else {
			nonNukableCount++
		}
	}

	query := QueryParams{}
	if r.query != nil {
		query = *r.query
	}

	output := InspectOutput{
		Timestamp: time.Now(),
		Command:   r.command,
		Query:     query,
		Resources: r.found,
		Errors:    r.errors,
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
	deletedCount := 0
	failedCount := 0
	for _, e := range r.deleted {
		if e.Success {
			deletedCount++
		} else {
			failedCount++
		}
	}

	output := NukeOutput{
		Timestamp: time.Now(),
		Command:   r.command,
		Regions:   r.regions,
		Resources: r.deleted,
		Errors:    r.errors,
		Summary: NukeSummary{
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
