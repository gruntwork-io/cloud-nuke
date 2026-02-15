package renderers

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"
)

// Emoji constants for CLI output
const (
	SuccessEmoji = "✅"
	FailureEmoji = "❌"
)

// MaxResourcesForDetailedTable is the threshold above which the CLI renders
// a summary table (grouped by resource type and region) instead of listing
// every individual resource. This prevents O(n²) pterm column-width
// calculation from hanging the CLI with very large resource counts.
const MaxResourcesForDetailedTable = 500

// CLIRenderer outputs results to terminal using pterm.
type CLIRenderer struct {
	writer      io.Writer
	spinner     *pterm.SpinnerPrinter
	progressBar *pterm.ProgressbarPrinter
	found       []reporting.ResourceFound
	deleted     []reporting.ResourceDeleted
	errors      []reporting.GeneralError
	nukeMode    bool // true if NukeStarted was received, determines if ScanComplete is terminal
}

// NewCLIRenderer creates a CLI renderer with an active spinner.
func NewCLIRenderer(writer io.Writer) *CLIRenderer {
	if writer == nil {
		writer = os.Stdout
	}
	spinner, err := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
	if err != nil {
		// Log but don't fail - spinner is non-critical UI element
		_, _ = fmt.Fprintf(writer, "Warning: failed to start spinner: %v\n", err)
	}
	return &CLIRenderer{
		writer:  writer,
		spinner: spinner,
		found:   make([]reporting.ResourceFound, 0),
		deleted: make([]reporting.ResourceDeleted, 0),
		errors:  make([]reporting.GeneralError, 0),
	}
}

// OnEvent routes events to appropriate handlers.
// Terminal events (ScanComplete for inspect, NukeComplete for nuke) trigger final output.
func (r *CLIRenderer) OnEvent(event reporting.Event) {
	switch e := event.(type) {
	case reporting.ScanStarted:
		r.handleScanStarted(e)
	case reporting.ScanProgress:
		r.updateSpinner(fmt.Sprintf("Scanning %s in %s", e.ResourceType, e.Region))
	case reporting.ResourceFound:
		r.found = append(r.found, e)
	case reporting.ScanComplete:
		r.handleScanComplete()
	case reporting.ResourceDeleted:
		r.handleResourceDeleted(e)
	case reporting.GeneralError:
		r.errors = append(r.errors, e)
	case reporting.NukeStarted:
		r.handleNukeStarted(e)
	case reporting.NukeProgress:
		r.updateProgressBar(fmt.Sprintf("Nuking batch of %d %s in %s", e.BatchSize, e.ResourceType, e.Region))
	case reporting.NukeComplete:
		r.handleNukeComplete()
	}
}

// updateSpinner safely updates spinner text if spinner is active.
func (r *CLIRenderer) updateSpinner(text string) {
	if r.spinner != nil {
		r.spinner.UpdateText(text)
	}
}

// updateProgressBar safely updates progress bar title if progress bar is active.
func (r *CLIRenderer) updateProgressBar(title string) {
	if r.progressBar != nil {
		r.progressBar.UpdateTitle(title)
	}
}

// handleScanComplete stops the spinner and displays found resources.
func (r *CLIRenderer) handleScanComplete() {
	if r.spinner != nil {
		_ = r.spinner.Stop()
		r.spinner = nil
	}
	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Found Resources")
	r.printFoundTable()
	// If not in nuke mode, ScanComplete is terminal - finalize output
	if !r.nukeMode {
		r.printErrorsTable()
		if len(r.errors) == 0 && len(r.found) == 0 {
			pterm.Info.WithWriter(r.writer).Println("No resources found.")
		}
	}
}

// handleResourceDeleted records deletion result and updates progress bar.
func (r *CLIRenderer) handleResourceDeleted(e reporting.ResourceDeleted) {
	r.deleted = append(r.deleted, e)
	if r.progressBar != nil {
		r.progressBar.Add(1)
	}
}

// handleNukeStarted initializes nuke mode and starts the progress bar.
func (r *CLIRenderer) handleNukeStarted(e reporting.NukeStarted) {
	r.nukeMode = true
	progressBar, err := pterm.DefaultProgressbar.WithTotal(e.Total).Start()
	if err != nil {
		_, _ = fmt.Fprintf(r.writer, "Warning: failed to start progress bar: %v\n", err)
	}
	r.progressBar = progressBar
}

// handleNukeComplete stops progress bar and displays final results.
func (r *CLIRenderer) handleNukeComplete() {
	if r.progressBar != nil {
		_, _ = r.progressBar.Stop()
		r.progressBar = nil
	}
	r.printErrorsTable()
	r.printDeletedTable()
}

func (r *CLIRenderer) printErrorsTable() {
	if len(r.errors) == 0 {
		return
	}

	// Workaround for pterm progressbar cleanup
	_, _ = r.writer.Write([]byte("\r"))

	tableData := pterm.TableData{
		{"Resource Type", "Description", "Error"},
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

	if len(r.found) > MaxResourcesForDetailedTable {
		r.printFoundSummaryTable()
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

// printFoundSummaryTable renders a compact summary grouped by resource type
// and region instead of listing every individual resource.
func (r *CLIRenderer) printFoundSummaryTable() {
	type key struct {
		ResourceType string
		Region       string
	}
	type counts struct {
		total      int
		nukable    int
		nonNukable int
	}

	summary := make(map[key]*counts)
	var order []key

	for _, e := range r.found {
		k := key{e.ResourceType, e.Region}
		c, exists := summary[k]
		if !exists {
			c = &counts{}
			summary[k] = c
			order = append(order, k)
		}
		c.total++
		if e.Nukable {
			c.nukable++
		} else {
			c.nonNukable++
		}
	}

	tableData := pterm.TableData{
		{"Resource Type", "Region", "Count", "Nukable", "Not Nukable"},
	}
	for _, k := range order {
		c := summary[k]
		tableData = append(tableData, []string{
			k.ResourceType,
			k.Region,
			fmt.Sprintf("%d", c.total),
			fmt.Sprintf("%d", c.nukable),
			fmt.Sprintf("%d", c.nonNukable),
		})
	}

	pterm.Info.WithWriter(r.writer).Printfln(
		"Showing summary (%d resources total). Use --output json for full details.",
		len(r.found),
	)

	_ = pterm.DefaultTable.
		WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		WithWriter(r.writer).
		Render()
}

// handleScanStarted displays query parameters and restarts spinner.
func (r *CLIRenderer) handleScanStarted(e reporting.ScanStarted) {
	// Stop spinner temporarily to print section header
	if r.spinner != nil {
		_ = r.spinner.Stop()
	}

	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Resource Query Parameters")

	tableData := pterm.TableData{
		{"Query Parameter", "Value"},
	}

	// Listing regions if there are <= 5 regions, otherwise the table format breaks
	if len(e.Regions) > 5 {
		tableData = append(tableData, []string{"Target Regions", fmt.Sprintf("%d regions (too many to list all)", len(e.Regions))})
	} else {
		tableData = append(tableData, []string{"Target Regions", strings.Join(e.Regions, ", ")})
	}

	// Listing resource types if there are <= 5 resources, otherwise the table format breaks
	if len(e.ResourceTypes) > 5 {
		tableData = append(tableData, []string{"Target Resource Types", fmt.Sprintf("%d resource types (too many to list all)", len(e.ResourceTypes))})
	} else {
		tableData = append(tableData, []string{"Target Resource Types", strings.Join(e.ResourceTypes, ", ")})
	}

	if e.ExcludeAfter != "" {
		tableData = append(tableData, []string{"Exclude After Filter", e.ExcludeAfter})
	}
	if e.IncludeAfter != "" {
		tableData = append(tableData, []string{"Include After Filter", e.IncludeAfter})
	}
	tableData = append(tableData, []string{"List Unaliased KMS Keys", fmt.Sprintf("%t", e.ListUnaliasedKMSKeys)})

	_ = pterm.DefaultTable.WithBoxed(true).
		WithData(tableData).
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		WithWriter(r.writer).
		Render()

	// Restart spinner
	spinner, err := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
	if err != nil {
		_, _ = fmt.Fprintf(r.writer, "Warning: failed to restart spinner: %v\n", err)
	}
	r.spinner = spinner
}

func (r *CLIRenderer) printDeletedTable() {
	if len(r.deleted) == 0 {
		return
	}

	// Workaround for pterm progressbar cleanup
	_, _ = r.writer.Write([]byte("\r"))

	if len(r.deleted) > MaxResourcesForDetailedTable {
		r.printDeletedSummaryTable()
		return
	}

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

// printDeletedSummaryTable renders a compact summary of deletion results
// grouped by resource type and region.
func (r *CLIRenderer) printDeletedSummaryTable() {
	type key struct {
		ResourceType string
		Region       string
	}
	type counts struct {
		success int
		failure int
	}

	summary := make(map[key]*counts)
	var order []key

	for _, e := range r.deleted {
		k := key{e.ResourceType, e.Region}
		c, exists := summary[k]
		if !exists {
			c = &counts{}
			summary[k] = c
			order = append(order, k)
		}
		if e.Success {
			c.success++
		} else {
			c.failure++
		}
	}

	tableData := pterm.TableData{
		{"Resource Type", "Region", "Successful", "Failed"},
	}
	for _, k := range order {
		c := summary[k]
		tableData = append(tableData, []string{
			k.ResourceType,
			k.Region,
			fmt.Sprintf("%d", c.success),
			fmt.Sprintf("%d", c.failure),
		})
	}

	pterm.Info.WithWriter(r.writer).Printfln(
		"Showing summary (%d resources deleted). Use --output json for full details.",
		len(r.deleted),
	)

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
