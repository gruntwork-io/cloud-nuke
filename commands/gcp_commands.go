package commands

import (
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

// GCP Command Handlers
// These functions implement the CLI commands for GCP operations

// gcpNuke is the main command handler for nuking (deleting) GCP resources.
// It supports time-based filtering and config file overrides.
//
// Note: The --resource-type and --exclude-resource-type flags are currently ignored for GCP.
// GCP always processes all resource types. This should be implemented in a future PR.
func gcpNuke(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("gcp")()

	// Handle the --list-resource-types flag
	if c.Bool(FlagListResourceTypes) {
		return handleListGcpResourceTypes()
	}

	// Parse and set log level
	if err := parseLogLevel(c); err != nil {
		return err
	}

	// Load config file if provided
	configObj, err := loadConfigFile(c.String(FlagConfig))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	projectID := c.String(FlagProjectID)

	// Apply timeout to config
	if err := parseAndApplyTimeout(c, &configObj); err != nil {
		return err
	}

	// Apply time filters to config
	if err := parseAndApplyTimeFilters(c, &configObj); err != nil {
		return err
	}

	// Get output preferences
	outputFormat := c.String(FlagOutputFormat)
	outputFile := c.String(FlagOutputFile)

	return gcpNukeHelper(c, configObj, projectID, outputFormat, outputFile)
}

// gcpInspect is the command handler for non-destructive inspection of GCP resources.
// It lists resources that would be deleted without actually deleting them.
//
// Note: The --resource-type and --exclude-resource-type flags are currently ignored for GCP.
// GCP always inspects all resource types. This should be implemented in a future PR.
func gcpInspect(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("gcp-inspect")()

	// Handle the --list-resource-types flag
	if c.Bool(FlagListResourceTypes) {
		return handleListGcpResourceTypes()
	}

	// Parse and set log level
	if err := parseLogLevel(c); err != nil {
		return err
	}

	projectID := c.String(FlagProjectID)
	configObj := config.Config{}

	// Apply time filters to config
	if err := parseAndApplyTimeFilters(c, &configObj); err != nil {
		return err
	}

	// Get output preferences
	outputFormat := c.String(FlagOutputFormat)
	outputFile := c.String(FlagOutputFile)

	// Retrieve and display resources without deleting them
	_, err := handleGetGcpResourcesWithFormat(c, configObj, projectID, outputFormat, outputFile)
	return err
}

// Helper Functions
// These functions contain shared logic used by multiple command handlers

// gcpNukeHelper is the core logic for nuking GCP resources.
// It retrieves resources, confirms deletion with the user, and executes the nuke operation.
func gcpNukeHelper(c *cli.Context, configObj config.Config, projectID string, outputFormat string, outputFile string) error {
	// Retrieve all matching resources
	account, err := handleGetGcpResourcesWithFormat(c, configObj, projectID, outputFormat, outputFile)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting resources",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	// Confirm with user before proceeding (unless --force or --dry-run is set)
	shouldProceed, err := confirmNuke(c, len(account.Resources) > 0)
	if err != nil {
		return err
	}

	// Execute the nuke operation if confirmed
	if shouldProceed {
		gcp.NukeAllResources(account, nil)
		ui.RenderRunReportWithFormat(outputFormat, outputFile)
	}

	return nil
}

// handleGetGcpResourcesWithFormat retrieves all GCP resources matching the filters and renders them
// in the specified output format. This is used for both inspect and nuke operations.
func handleGetGcpResourcesWithFormat(c *cli.Context, configObj config.Config, projectID string, outputFormat string, outputFile string) (
	*gcp.GcpProjectResources, error) {
	// Display query parameters (only for table format to avoid cluttering JSON output)
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("GCP Resource Query Parameters")
		pterm.Println("Project ID:", projectID)
		pterm.Println()
	}

	// Retrieve all resources matching the filters
	// Note: GCP uses config-based time filtering instead of query-based like AWS
	accountResources, err := gcp.GetAllResources(projectID, configObj, time.Time{}, time.Time{})
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return nil, errors.WithStackTrace(err)
	}

	// Display found resources header (only for table format)
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Found GCP Resources")
	}

	// Render the resources in the requested format (table or JSON)
	err = ui.RenderGcpResourcesAsTableWithFormat(accountResources, outputFormat, outputFile)

	return accountResources, err
}

// handleListGcpResourceTypes displays all available GCP resource types that can be targeted.
func handleListGcpResourceTypes() error {
	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("GCP Resource Types")
	return ui.RenderResourceTypesAsBulletList(gcp.ListResourceTypes())
}
