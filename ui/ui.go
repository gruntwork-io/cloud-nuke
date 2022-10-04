package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/pterm/pterm"
)

func PrintRunReport(w io.Writer) {
	// Workaround an issue where the pterm progressbar might not be cleaned up correctly
	w.Write([]byte("\r"))

	// Records is a map[string]Entry from the report package. This maps contains an entry for
	// every AWS resource a given run of cloud-nuke operated on, along with the result (error or nil)
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
			//
			// We intentionally truncate the error message to its first 40 characters, because pterm tables are not fully
			// responsive within a terminal, and this truncation allows rows to situate nicely within the table
			//
			// If we upgrade to a library that can render flexbox tables in the terminal we should revisit this
			errSymbol = fmt.Sprintf("❌ %s", truncate(removeNewlines(entry.Error.Error()), 40))
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

// removeNewlines will delete all the newlines in a given string, which is useful for making error messages
// "sit" more nicely within their specified table cells in the terminal
func removeNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", "")
}
