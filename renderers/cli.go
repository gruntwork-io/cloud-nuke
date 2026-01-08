package renderers

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/pterm/pterm"
)

const (
	SuccessEmoji = "✅"
	FailureEmoji = "❌"
)

// NukeCLIRenderer outputs nuke results to terminal matching existing cloud-nuke format.
// Collects deletion results and renders final table summary.
// Note: Progress bar is handled by aws.NukeAllResources/gcp.NukeAllResources, not this renderer.
type NukeCLIRenderer struct {
	writer    io.Writer
	resources []nukeResult
	errors    []generalError
}

type nukeResult struct {
	Identifier   string
	ResourceType string
	Success      bool
	Error        string
}

type generalError struct {
	ResourceType string
	Description  string
	Error        string
}

// NewNukeCLIRenderer creates a nuke CLI renderer.
// The writer parameter specifies where the final table output will be written.
func NewNukeCLIRenderer(writer io.Writer) *NukeCLIRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	return &NukeCLIRenderer{
		writer:    writer,
		resources: make([]nukeResult, 0),
		errors:    make([]generalError, 0),
	}
}

// OnEvent collects deletion results for final table output.
// Progress bar updates are handled by aws.NukeAllResources/gcp.NukeAllResources.
func (r *NukeCLIRenderer) OnEvent(event reporting.Event) {
	switch e := event.(type) {
	case reporting.ResourceDeleted:
		r.resources = append(r.resources, nukeResult{
			Identifier:   e.Identifier,
			ResourceType: e.ResourceType,
			Success:      e.Success,
			Error:        e.Error,
		})

	case reporting.GeneralError:
		r.errors = append(r.errors, generalError{
			ResourceType: e.ResourceType,
			Description:  e.Description,
			Error:        e.Error,
		})
	}
}

// Render outputs final nuke report (matches existing cloud-nuke format)
func (r *NukeCLIRenderer) Render() error {
	r.printErrors(r.errors)
	r.printRunReport(r.resources)
	return nil
}

func (r *NukeCLIRenderer) printErrors(errors []generalError) {
	if len(errors) == 0 {
		return
	}

	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	_, _ = r.writer.Write([]byte("\r"))

	tableData := pterm.TableData{
		{"ResourceType", "Description", "Error"},
	}
	for _, err := range errors {
		tableData = append(tableData, []string{err.ResourceType, err.Description, err.Error})
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithBoxed(true).
		WithRowSeparator("-").
		WithLeftAlignment().
		WithData(tableData).
		WithWriter(r.writer).
		Render()

	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	_, _ = r.writer.Write([]byte("\r"))
}

func (r *NukeCLIRenderer) printRunReport(resources []nukeResult) {
	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	_, _ = r.writer.Write([]byte("\r"))

	if len(resources) == 0 {
		pterm.Info.WithWriter(r.writer).Println("No resources touched in this run.")
		return
	}

	tableData := pterm.TableData{
		{"Identifier", "Resource Type", "Deleted Successfully"},
	}

	for _, res := range resources {
		var status string
		if res.Success {
			status = SuccessEmoji
		} else {
			status = fmt.Sprintf("%s %s", FailureEmoji, truncate(removeNewlines(res.Error), 40))
		}
		tableData = append(tableData, []string{res.Identifier, res.ResourceType, status})
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithBoxed(true).
		WithRowSeparator("-").
		WithLeftAlignment().
		WithData(tableData).
		WithWriter(r.writer).
		Render()

	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	_, _ = r.writer.Write([]byte("\r"))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func removeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}

// InspectCLIRenderer renders inspection results as a table.
// Table columns: Resource Type | Region | Identifier | Nukable
type InspectCLIRenderer struct {
	writer    io.Writer
	resources []inspectResult
}

type inspectResult struct {
	ResourceType string
	Region       string
	Identifier   string
	Nukable      bool
	Reason       string
}

// NewInspectCLIRenderer creates an inspect renderer
func NewInspectCLIRenderer(writer io.Writer) *InspectCLIRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	return &InspectCLIRenderer{
		writer:    writer,
		resources: make([]inspectResult, 0),
	}
}

// OnEvent buffers discovery events
func (r *InspectCLIRenderer) OnEvent(event reporting.Event) {
	if e, ok := event.(reporting.ResourceFound); ok {
		r.resources = append(r.resources, inspectResult{
			ResourceType: e.ResourceType,
			Region:       e.Region,
			Identifier:   e.Identifier,
			Nukable:      e.Nukable,
			Reason:       e.Reason,
		})
	}
}

// Render outputs inspection table
func (r *InspectCLIRenderer) Render() error {
	if len(r.resources) == 0 {
		fmt.Fprintln(r.writer, "No resources found.")
		return nil
	}

	tableData := pterm.TableData{
		{"Resource Type", "Region", "Identifier", "Nukable"},
	}

	for _, res := range r.resources {
		nukable := SuccessEmoji
		if !res.Nukable {
			nukable = res.Reason
		}
		tableData = append(tableData, []string{res.ResourceType, res.Region, res.Identifier, nukable})
	}

	return pterm.DefaultTable.
		WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		WithWriter(r.writer).
		Render()
}
