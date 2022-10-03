package report

import (
	"fmt"
	"io"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/progressbar"
	"github.com/pterm/pterm"
)

var m = &sync.Mutex{}

var records = make(map[string]Entry)

func Record(e Entry) {
	defer m.Unlock()
	m.Lock()
	records[e.Identifier] = e
	// Increment the progressbar so the user feels measurable progress on long-running nuke jobs
	p := progressbar.GetProgressbar()
	p.Increment()
}

// RecordBatch accepts a BatchEntry that contains a slice of identifiers, loops through them and converts each identifier to
// a standard Entry. This is useful for supporting batch delete workflows in cloud-nuke (such as cloudwatch_dashboards)
func RecordBatch(e BatchEntry) {
	for _, identifier := range e.Identifiers {
		entry := Entry{
			Identifier:   identifier,
			ResourceType: e.ResourceType,
			Error:        e.Error,
		}
		Record(entry)
	}
}

func Print(w io.Writer) {
	// Start by removing the progressbar, now that we're ready to display the table report
	p := progressbar.GetProgressbar()
	p.Stop()
	fmt.Println()
	fmt.Println()

	data := make([][]string, len(records))
	entriesToDisplay := []Entry{}
	for _, entry := range records {
		entriesToDisplay = append(entriesToDisplay, entry)
	}

	r, _ := glamour.NewTermRenderer(
		// detect background color and pick either the default dark or light theme
		glamour.WithAutoStyle(),
		// wrap output at specific width
		glamour.WithWordWrap(40),
	)

	for idx, entry := range entriesToDisplay {
		var errSymbol string
		if entry.Error != nil {
			errStr := entry.Error.Error()
			renderedStr, renderErr := r.Render(errStr)
			if renderErr != nil {
				fmt.Errorf("Error rendering report cell: %s\n", renderErr)
			}
			// If we encountered an error when deleting the resource, display it in-line within the table for the operator
			errSymbol = fmt.Sprintf("  ❌ %s\n\n   ", renderedStr)
		} else {
			errSymbol = "     ✅    "
		}
		data[idx] = []string{entry.Identifier, entry.ResourceType, errSymbol}
	}

	renderTableWithHeader([]string{"Identifier", "Resource Type", "Deleted Successfully"}, data, w)
}

func renderSection(sectionTitle string, w io.Writer) {
	section := pterm.DefaultSection.WithStyle(pterm.NewStyle(pterm.FgLightCyan))
	section = section.WithWriter(w).WithLevel(0)
	section.Println(sectionTitle)
}

func renderTableWithHeader(headers []string, data [][]string, w io.Writer) {
	tableData := pterm.TableData{
		headers,
	}
	for idx := range data {
		tableData = append(tableData, data[idx])
	}
	renderErr := pterm.DefaultTable.
		WithHasHeader().
		WithBoxed(true).
		WithRowSeparator("-").
		WithData(tableData).
		WithWriter(w).
		Render()

	if renderErr != nil {
		logging.Logger.Infof("Error rendering report table: %s\n", renderErr)
	}
}

func Reset() {
	records = make(map[string]Entry)
}

// Custom types

type Entry struct {
	Identifier   string
	ResourceType string
	Error        error
}

type BatchEntry struct {
	Identifiers  []string
	ResourceType string
	Error        error
}
