package ui

import (
	"fmt"
	"io"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/pterm/pterm"
)

func PrintRunReport(w io.Writer) {
	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	w.Write([]byte("\r"))

	records := report.GetRecords()

	data := make([][]string, len(records))
	entriesToDisplay := []report.Entry{}
	for _, entry := range records {
		entriesToDisplay = append(entriesToDisplay, entry)
	}

	for idx, entry := range entriesToDisplay {
		var errSymbol string
		if entry.Error != nil {
			// If we encountered an error when deleting the resource, display it in-line within the table for the operator
			errSymbol = fmt.Sprintf("❌ %s", truncate(entry.Error.Error(), 40))
		} else {
			errSymbol = "✅"
		}
		data[idx] = []string{entry.Identifier, entry.ResourceType, errSymbol}
	}

	renderTableWithHeader([]string{"Identifier", "Resource Type", "Deleted Successfully"}, data, w)

	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	w.Write([]byte("\r"))
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
		WithLeftAlignment().
		WithData(tableData).
		WithWriter(w).
		Render()

	if renderErr != nil {
		logging.Logger.Infof("Error rendering report table: %s\n", renderErr)
	}
}

// truncate accepts a string and a max length. If the max length is less than the string's current length,
// then only the first maxLen characters of the string are returned
func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
