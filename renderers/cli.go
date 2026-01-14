package renderers

import (
	"fmt"
	"io"
	"os"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"
)

const (
	SuccessEmoji = "✅"
	FailureEmoji = "❌"
)

// CLIRenderer outputs results to terminal.
// Renders whatever events were collected.
// Handles progress events in real-time (spinner) and result events at the end (tables).
type CLIRenderer struct {
	writer  io.Writer
	spinner *pterm.SpinnerPrinter
	found   []reporting.ResourceFound
	deleted []reporting.ResourceDeleted
	errors  []reporting.GeneralError
}

// NewCLIRenderer creates a CLI renderer with an active spinner
func NewCLIRenderer(writer io.Writer) *CLIRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	spinner, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
	return &CLIRenderer{
		writer:  writer,
		spinner: spinner,
		found:   make([]reporting.ResourceFound, 0),
		deleted: make([]reporting.ResourceDeleted, 0),
		errors:  make([]reporting.GeneralError, 0),
	}
}

// OnEvent handles events - progress events update spinner, result events are collected
func (r *CLIRenderer) OnEvent(event reporting.Event) {
	switch e := event.(type) {
	// Progress event - update spinner text
	case reporting.ScanProgress:
		r.spinner.UpdateText(fmt.Sprintf("Searching %s resources in %s", e.ResourceType, e.Region))
	// Result events - collect for final render
	case reporting.ResourceFound:
		r.found = append(r.found, e)
	case reporting.ResourceDeleted:
		r.deleted = append(r.deleted, e)
	case reporting.GeneralError:
		r.errors = append(r.errors, e)
	}
}

// Render outputs all collected data
func (r *CLIRenderer) Render() error {
	// Stop the spinner
	_ = r.spinner.Stop()

	r.printErrors()
	r.printFoundTable()
	r.printDeletedTable()

	if len(r.errors) == 0 && len(r.found) == 0 && len(r.deleted) == 0 {
		_, _ = fmt.Fprintln(r.writer, "No resources found.")
	}
	return nil
}

func (r *CLIRenderer) printErrors() {
	if len(r.errors) == 0 {
		return
	}

	_, _ = r.writer.Write([]byte("\r"))

	tableData := pterm.TableData{
		{"ResourceType", "Description", "Error"},
	}
	for _, e := range r.errors {
		tableData = append(tableData, []string{e.ResourceType, e.Description, e.Error})
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithBoxed(true).
		WithRowSeparator("-").
		WithLeftAlignment().
		WithData(tableData).
		WithWriter(r.writer).
		Render()

	_, _ = r.writer.Write([]byte("\r"))
}

func (r *CLIRenderer) printFoundTable() {
	if len(r.found) == 0 {
		return
	}

	tableData := pterm.TableData{
		{"Resource Type", "Region", "Identifier", "Nukable"},
	}

	for _, e := range r.found {
		nukable := SuccessEmoji
		if !e.Nukable {
			nukable = e.Reason
		}
		tableData = append(tableData, []string{e.ResourceType, e.Region, e.Identifier, nukable})
	}

	_ = pterm.DefaultTable.
		WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		WithWriter(r.writer).
		Render()
}

func (r *CLIRenderer) printDeletedTable() {
	if len(r.deleted) == 0 {
		return
	}

	_, _ = r.writer.Write([]byte("\r"))

	tableData := pterm.TableData{
		{"Identifier", "Resource Type", "Deleted Successfully"},
	}

	for _, e := range r.deleted {
		var status string
		if e.Success {
			status = SuccessEmoji
		} else {
			status = fmt.Sprintf("%s %s", FailureEmoji, util.Truncate(util.RemoveNewlines(e.Error), 40))
		}
		tableData = append(tableData, []string{e.Identifier, e.ResourceType, status})
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithBoxed(true).
		WithRowSeparator("-").
		WithLeftAlignment().
		WithData(tableData).
		WithWriter(r.writer).
		Render()

	_, _ = r.writer.Write([]byte("\r"))
}
