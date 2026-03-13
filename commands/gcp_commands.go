package commands

import (
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp"
	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/urfave/cli/v2"
)

// GCP Command Handlers
// These functions implement the CLI commands for GCP operations

// gcpNuke is the main command handler for nuking (deleting) GCP resources.
// It supports region filtering, resource type filtering, time-based filtering,
// and config file overrides.
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

	query := &gcp.Query{
		ProjectID:            c.String(FlagProjectID),
		Regions:              c.StringSlice(FlagRegion),
		ExcludeRegions:       c.StringSlice(FlagExcludeRegion),
		ResourceTypes:        c.StringSlice(FlagResourceType),
		ExcludeResourceTypes: c.StringSlice(FlagExcludeResourceType),
	}

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

	if err := query.Validate(); err != nil {
		return err
	}

	return gcpNukeHelper(c, configObj, query, outputFormat, outputFile)
}

// gcpInspect is the command handler for non-destructive inspection of GCP resources.
// It lists resources that would be deleted without actually deleting them.
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

	query := &gcp.Query{
		ProjectID:            c.String(FlagProjectID),
		Regions:              c.StringSlice(FlagRegion),
		ExcludeRegions:       c.StringSlice(FlagExcludeRegion),
		ResourceTypes:        c.StringSlice(FlagResourceType),
		ExcludeResourceTypes: c.StringSlice(FlagExcludeResourceType),
	}

	// Load config file if provided
	configObj, err := loadConfigFile(c.String(FlagConfig))
	if err != nil {
		return errors.WithStackTrace(err)
	}

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

	if err := query.Validate(); err != nil {
		return err
	}

	// Retrieve and display resources without deleting them
	_, err = handleGetGcpResourcesWithFormat(c, configObj, query, outputFormat, outputFile)
	return err
}

// Helper Functions
// These functions contain shared logic used by multiple command handlers

// gcpNukeHelper is the core logic for nuking GCP resources.
// It retrieves resources, confirms deletion with the user, and executes the nuke operation.
func gcpNukeHelper(c *cli.Context, configObj config.Config, query *gcp.Query, outputFormat string, outputFile string) error {
	// Setup reporting - cleanup calls Complete() and closes writer
	collector, cleanup, err := setupGcpReporting(outputFormat, outputFile, query.ProjectID)
	if err != nil {
		return err
	}
	defer cleanup()

	// Retrieve all matching resources (emits ResourceFound events via collector)
	account, err := gcp.GetAllResources(c.Context, query, configObj, collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting resources",
		}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	// Signal scan complete - renderer will show found resources table
	collector.Emit(reporting.ScanComplete{})

	// Confirm with user before proceeding (unless --force or --dry-run is set)
	shouldProceed, err := confirmNuke(c, len(account.Resources) > 0)
	if err != nil {
		return err
	}

	// Execute the nuke operation if confirmed
	if shouldProceed {
		if err := gcp.NukeAllResources(c.Context, account, query.Regions, collector); err != nil {
			return err
		}
	}

	return nil
}

// handleGetGcpResourcesWithFormat retrieves all GCP resources matching the filters and renders them
// in the specified output format. This is used for inspect operations only.
func handleGetGcpResourcesWithFormat(c *cli.Context, configObj config.Config, query *gcp.Query, outputFormat string, outputFile string) (
	*gcp.GcpProjectResources, error) {
	// Setup reporting - cleanup calls Complete() and closes writer
	collector, cleanup, err := setupGcpReporting(outputFormat, outputFile, query.ProjectID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Retrieve all resources matching the filters (emits ResourceFound events via collector)
	accountResources, err := gcp.GetAllResources(c.Context, query, configObj, collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return nil, errors.WithStackTrace(err)
	}

	// Signal scan complete - renderer will show found resources table
	collector.Emit(reporting.ScanComplete{})

	return accountResources, nil
}

// handleListGcpResourceTypes displays all available GCP resource types that can be targeted.
func handleListGcpResourceTypes() error {
	return printResourceTypes("GCP Resource Types", gcp.ListResourceTypes())
}

// setupGcpReporting creates a collector and appropriate renderer for GCP operations.
// Returns the collector, cleanup function (which calls Complete() and closes writer), and any error.
func setupGcpReporting(outputFormat string, outputFile string, projectID string) (
	*reporting.Collector, func(), error) {
	return setupReporting(outputFormat, outputFile, renderers.JSONRendererConfig{
		Command: "gcp",
		Regions: []string{projectID},
	})
}
