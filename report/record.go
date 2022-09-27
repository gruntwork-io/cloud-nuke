package report

import (
	"fmt"
	"io"
	"sync"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/pterm/pterm"
)

var m = &sync.Mutex{}

var records = make(map[string]Entry)

func Record(e Entry) {
	defer m.Unlock()
	m.Lock()
	records[e.Identifier] = e
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
	renderSection("Nuking complete:", w)
	data := make([][]string, len(records))
	entriesToDisplay := []Entry{}
	for _, entry := range records {
		entriesToDisplay = append(entriesToDisplay, entry)
	}

	for idx, entry := range entriesToDisplay {
		var errSymbol string
		if entry.Error != nil {
			errSymbol = fmt.Sprintf("  ❌ %s   ", entry.Error.Error())
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
