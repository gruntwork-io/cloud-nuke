package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andrewderr/cloud-nuke-a1/aws"
	"github.com/gruntwork-io/go-commons/errors"

	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/progressbar"
	"github.com/andrewderr/cloud-nuke-a1/report"
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

	// Short-circuit if there are no entries to display
	if len(records) == 0 {
		pterm.Info.Println("No resources touched in this run.")
		return
	}

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
		logging.Infof("Error rendering report table: %s\n", renderErr)
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

func RenderResourcesAsTable(account *aws.AwsAccountResources) error {
	var tableData [][]string
	tableData = append(tableData, []string{"Resource Type", "Region", "Identifier"})

	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			for _, identifier := range (*foundResources).ResourceIdentifiers() {
				tableData = append(tableData, []string{(*foundResources).ResourceName(), region, identifier})
			}
		}
	}

	return pterm.DefaultTable.WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		Render()
}

func RenderResourceTypesAsBulletList(resourceTypes []string) error {
	var items []pterm.BulletListItem
	for _, resourceType := range resourceTypes {
		items = append(items, pterm.BulletListItem{Level: 0, Text: resourceType})
	}

	return pterm.DefaultBulletList.WithItems(items).Render()
}

func RenderQueryAsBulletList(query *aws.Query) error {
	var tableData [][]string
	tableData = append(tableData, []string{"Query Parameter", "Value"})

	// Listing regions if there are <= 5 regions, otherwise the table format breaks.
	if len(query.Regions) > 5 {
		tableData = append(tableData,
			[]string{"Target Regions", fmt.Sprintf("%d regions (too many to list all)", len(query.Regions))})
	} else {
		tableData = append(tableData, []string{"Target Regions", strings.Join(query.Regions, ", ")})
	}

	// Listing resource types if there are <= 5 resources, otherwise the table format breaks.
	if len(query.ResourceTypes) > 5 {
		tableData = append(tableData,
			[]string{"Target Resource Types", fmt.Sprintf("%d resource types (too many to list all)", len(query.ResourceTypes))})
	} else {
		tableData = append(tableData, []string{"Target Resource Types", strings.Join(query.ResourceTypes, ", ")})
	}

	if query.ExcludeAfter != nil {
		tableData = append(tableData, []string{"Exclude After Filter", query.ExcludeAfter.Format("2006-01-02 15:04:05")})
	}
	if query.IncludeAfter != nil {
		tableData = append(tableData, []string{"Include After Filter", query.IncludeAfter.Format("2006-01-02 15:04:05")})
	}
	tableData = append(tableData, []string{"List Unaliased KMS Keys", fmt.Sprintf("%t", query.ListUnaliasedKMSKeys)})

	return pterm.DefaultTable.WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		Render()
}

func RenderNukeConfirmationPrompt(prompt string, numRetryCount int) (bool, error) {
	prompts := 0

	pterm.Println()
	pterm.Warning.Println("THE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

	for prompts < numRetryCount {
		confirmPrompt := pterm.DefaultInteractiveTextInput.WithMultiLine(false)
		input, err := confirmPrompt.Show(prompt)
		if err != nil {
			logging.Errorf("[Failed to render prompt] %s", err)
			return false, errors.WithStackTrace(err)
		}

		response := strings.ToLower(strings.TrimSpace(input))
		if response == "nuke" {
			pterm.Println()
			return true, nil
		}

		pterm.Println()
		pterm.Error.Println(fmt.Sprintf("Invalid value was entered: %s. Try again.", input))
		prompts++
	}

	pterm.Println()
	return false, nil
}
