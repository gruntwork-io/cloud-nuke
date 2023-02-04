package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/progressbar"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/pterm/pterm"
)

// RenderRunReport should be called at the end of a support cloud-nuke function
// It will print a table showing resources were deleted, and what errors occurred
// Note that certain functions don't support the report table, such as aws-inspect,
// which prints its own findings out directly to os.Stdout
func RenderRunReport() {
	// Remove the progressbar, now that we're ready to display the table report
	p := progressbar.GetProgressbar()
	// This next entry is necessary to workaround an issue where the spinner is not reliably cleaned up before the
	// final run report table is printed
	fmt.Print("\r")
	p.Stop()
	pterm.Println()

	// Conditionally print the general error report, if in fact there were errors
	PrintGeneralErrorReport(os.Stdout)

	// Print the report showing the user what happened with each resource
	PrintRunReport(os.Stdout)
}

func PrintGeneralErrorReport(w io.Writer) {
	// generalErrors is a map[string]GeneralError from the report package. This map contains
	// an entry for every general error (that is, not a resource-specific erorr) that occurred
	// during a cloud-nuke run. A GeneralError, for example, would be when cloud-nuke fails to
	// look up any particular resource type due to a network blip, AWS API 500, etc
	generalErrors := report.GetErrors()

	// Only render the general error table if there are, indeed, general errors
	if len(generalErrors) > 0 {

		// Workaround an issue where the pterm progressbar might not be cleaned up correctly
		w.Write([]byte("\r"))

		data := make([][]string, len(generalErrors))
		entriesToDisplay := []report.GeneralError{}
		for _, generalErr := range generalErrors {
			entriesToDisplay = append(entriesToDisplay, generalErr)
		}

		for idx, generalErr := range entriesToDisplay {
			data[idx] = []string{generalErr.ResourceType, generalErr.Description, generalErr.Error.Error()}
		}

		renderTableWithHeader([]string{"ResourceType", "Description", "Error"}, data, w)

		// Workaround an issue where the pterm progressbar might not be cleaned up correctly
		w.Write([]byte("\r"))

	}
}

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
			errSymbol = fmt.Sprintf("%s %s", FailureEmoji, truncate(removeNewlines(entry.Error.Error()), 40))
		} else {
			errSymbol = SuccessEmoji
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
