package ui

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/pterm/pterm"
)

// MaxResourcesForDetailedTable is the threshold above which we switch from
// detailed per-resource table to a summary table showing counts by resource type.
// This prevents performance issues with very large resource counts where pterm
// table rendering becomes extremely slow.
// Use JSON output format (--output json) to get the complete list.
const MaxResourcesForDetailedTable = 500

// RenderRunReport should be called at the end of a support cloud-nuke function
// It will print a table showing resources were deleted, and what errors occurred
// Note that certain functions don't support the report table, such as aws-inspect,
// which prints its own findings out directly to os.Stdout
func RenderRunReport() {
	RenderRunReportWithFormat("table", "")
}

// RenderRunReportWithFormat renders the run report in the specified format
func RenderRunReportWithFormat(outputFormat string, outputFile string) {
	writer, closer, err := GetOutputWriter(outputFile)
	if err != nil {
		logging.Errorf("Failed to open output file: %s", err)
		return
	}
	defer func() {
		if err := closer(); err != nil {
			logging.Errorf("Failed to close output writer: %s", err)
		}
	}()

	if outputFormat == "json" {
		if err := RenderNukeReportAsJSON(writer); err != nil {
			logging.Errorf("Failed to render JSON report: %s", err)
		}
		return
	}

	// Table format output
	// Conditionally print the general error report, if in fact there were errors
	PrintGeneralErrorReport(writer)

	// Print the report showing the user what happened with each resource
	PrintRunReport(writer)
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
			errSymbol = fmt.Sprintf("%s %s", FailureEmoji, util.Truncate(
				util.RemoveNewlines(entry.Error.Error()), 40))
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

func RenderResourcesAsTable(account *aws.AwsAccountResources) error {
	return RenderResourcesAsTableWithFormat(account, nil, "table", "")
}

// RenderResourcesAsTableWithFormat renders resources in the specified format
func RenderResourcesAsTableWithFormat(account *aws.AwsAccountResources, query *aws.Query, outputFormat string, outputFile string) error {
	writer, closer, err := GetOutputWriter(outputFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := closer(); err != nil {
			logging.Errorf("Failed to close output writer: %s", err)
		}
	}()

	if outputFormat == "json" {
		return RenderInspectAsJSON(account, query, writer)
	}

	totalResources := account.TotalResourceCount()

	// For large resource counts, render a summary table instead of detailed table
	// to prevent performance issues with pterm table rendering
	if totalResources > MaxResourcesForDetailedTable {
		summary := buildResourceSummary(account.Resources)
		return renderSummaryTable(summary, totalResources, writer)
	}

	// Table format output for smaller datasets
	var tableData [][]string
	tableData = append(tableData, []string{"Resource Type", "Region", "Identifier", "Nukable"})

	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			for _, identifier := range (*foundResources).ResourceIdentifiers() {
				isnukable := SuccessEmoji
				_, err := (*foundResources).IsNukable(identifier)
				if err != nil {
					isnukable = err.Error()
				}

				tableData = append(tableData, []string{(*foundResources).ResourceName(), region, identifier, isnukable})
			}
		}
	}

	return renderPtermTable(tableData, writer)
}

// buildResourceSummary builds a summary map from AWS resources: map[resourceType]map[region]count
func buildResourceSummary(resources map[string]aws.AwsResources) map[string]map[string]int {
	summary := make(map[string]map[string]int)
	for region, resourcesInRegion := range resources {
		for _, foundResources := range resourcesInRegion.Resources {
			resourceType := (*foundResources).ResourceName()
			count := len((*foundResources).ResourceIdentifiers())
			if count == 0 {
				continue
			}
			if summary[resourceType] == nil {
				summary[resourceType] = make(map[string]int)
			}
			summary[resourceType][region] = count
		}
	}
	return summary
}

// renderSummaryTable renders a compact summary table showing resource counts by type and region
func renderSummaryTable(summary map[string]map[string]int, totalResources int, writer io.Writer) error {
	// Sort resource types for consistent output
	var resourceTypes []string
	for rt := range summary {
		resourceTypes = append(resourceTypes, rt)
	}
	sort.Strings(resourceTypes)

	// Build summary table
	var tableData [][]string
	tableData = append(tableData, []string{"Resource Type", "Region", "Count"})

	for _, resourceType := range resourceTypes {
		regions := summary[resourceType]

		// Sort regions for consistent output
		var regionNames []string
		for r := range regions {
			regionNames = append(regionNames, r)
		}
		sort.Strings(regionNames)

		for _, region := range regionNames {
			tableData = append(tableData, []string{resourceType, region, fmt.Sprintf("%d", regions[region])})
		}
	}

	// Print warning about summary mode
	pterm.Warning.Printfln(
		"Found %d resources (exceeds display limit of %d). Showing summary by resource type.",
		totalResources, MaxResourcesForDetailedTable)
	pterm.Info.Println("Use --output json to get the complete list of resources.")
	pterm.Println()

	return renderPtermTable(tableData, writer)
}

// renderPtermTable renders a pterm table to the given writer
func renderPtermTable(tableData [][]string, writer io.Writer) error {
	table := pterm.DefaultTable.WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-")

	if f, ok := writer.(*os.File); ok && (f == os.Stdout || f == os.Stderr) {
		return table.Render()
	}
	return table.WithWriter(writer).Render()
}

func RenderGcpResourcesAsTable(account *gcp.GcpProjectResources) error {
	return RenderGcpResourcesAsTableWithFormat(account, "table", "")
}

// RenderGcpResourcesAsTableWithFormat renders GCP resources in the specified format
func RenderGcpResourcesAsTableWithFormat(account *gcp.GcpProjectResources, outputFormat string, outputFile string) error {
	writer, closer, err := GetOutputWriter(outputFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := closer(); err != nil {
			logging.Errorf("Failed to close output writer: %s", err)
		}
	}()

	if outputFormat == "json" {
		return RenderGcpInspectAsJSON(account, writer)
	}

	totalResources := account.TotalResourceCount()

	// For large resource counts, render a summary table instead of detailed table
	// to prevent performance issues with pterm table rendering
	if totalResources > MaxResourcesForDetailedTable {
		summary := buildGcpResourceSummary(account.Resources)
		return renderSummaryTable(summary, totalResources, writer)
	}

	// Table format output for smaller datasets
	var tableData [][]string
	tableData = append(tableData, []string{"Resource Type", "Region", "Identifier", "Nukable"})

	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			for _, identifier := range (*foundResources).ResourceIdentifiers() {
				isnukable := SuccessEmoji
				_, err := (*foundResources).IsNukable(identifier)
				if err != nil {
					isnukable = err.Error()
				}

				tableData = append(tableData, []string{(*foundResources).ResourceName(), region, identifier, isnukable})
			}
		}
	}

	return renderPtermTable(tableData, writer)
}

// buildGcpResourceSummary builds a summary map from GCP resources: map[resourceType]map[region]count
func buildGcpResourceSummary(resources map[string]gcp.GcpResources) map[string]map[string]int {
	summary := make(map[string]map[string]int)
	for region, resourcesInRegion := range resources {
		for _, foundResources := range resourcesInRegion.Resources {
			resourceType := (*foundResources).ResourceName()
			count := len((*foundResources).ResourceIdentifiers())
			if count == 0 {
				continue
			}
			if summary[resourceType] == nil {
				summary[resourceType] = make(map[string]int)
			}
			summary[resourceType][region] = count
		}
	}
	return summary
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
