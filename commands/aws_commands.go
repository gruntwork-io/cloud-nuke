package commands

import (
	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/urfave/cli/v2"
)

// AWS Command Handlers
// These functions implement the CLI commands for AWS operations

// awsNuke is the main command handler for nuking (deleting) AWS resources.
// It supports resource filtering, time-based filtering, and config file overrides.
func awsNuke(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("aws")()

	// Handle the --list-resource-types flag
	if c.Bool(FlagListResourceTypes) {
		return handleListResourceTypes()
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

	// Apply timeout to config (consistent with GCP behavior)
	if err = parseAndApplyTimeout(c, &configObj); err != nil {
		return err
	}

	// Build AWS query from CLI flags
	query, err := generateQuery(c, c.Bool(FlagDeleteUnaliasedKMSKeys), nil, false)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Get output preferences
	outputFormat := c.String(FlagOutputFormat)
	outputFile := c.String(FlagOutputFile)

	return awsNukeHelper(c, configObj, query, outputFormat, outputFile)
}

// awsDefaults is the command handler for nuking AWS default VPCs and security groups.
// This is a specialized version of awsNuke that targets only default resources.
func awsDefaults(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("aws-defaults")()

	// Parse and set log level
	if err := parseLogLevel(c); err != nil {
		return err
	}

	// Determine which default resources to target based on flags.
	// Default VPCs have dependencies that must be deleted first.
	// The resource ordering in resource_registry.go ensures correct deletion order.
	resourceTypes := []string{
		"ec2-endpoint",      // Delete VPC endpoints in default VPCs
		"nat-gateway",       // Delete NAT gateways in default VPCs
		"network-interface", // Delete ENIs in default VPCs
		"internet-gateway",  // Detach and delete IGWs attached to default VPCs
		"ec2-subnet",        // Delete default subnets
		"vpc",               // Delete default VPCs (SGs and NACLs auto-deleted)
	}
	if c.Bool(FlagSGOnly) {
		resourceTypes = []string{"security-group"} // Only target default security groups
	}

	// Build query for default resources only
	query, err := generateQuery(c, false, resourceTypes, true)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Note: Config file feature is only available for the 'aws' command
	// The 'defaults-aws' command always uses table format output to stdout
	return awsNukeHelper(c, config.Config{}, query, DefaultOutputFormat, "")
}

// awsInspect is the command handler for non-destructive inspection of AWS resources.
// It lists resources that would be deleted without actually deleting them.
func awsInspect(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("aws-inspect")()

	// Handle the --list-resource-types flag
	if c.Bool(FlagListResourceTypes) {
		return handleListResourceTypes()
	}

	// Build AWS query from CLI flags
	query, err := generateQuery(c, c.Bool(FlagListUnaliasedKMSKeys), nil, false)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Get output preferences
	outputFormat := c.String(FlagOutputFormat)
	outputFile := c.String(FlagOutputFile)

	// Retrieve and display resources without deleting them
	_, err = handleGetResourcesWithFormat(c, config.Config{}, query, outputFormat, outputFile)
	return err
}

// Helper Functions
// These functions contain shared logic used by multiple command handlers

// awsNukeHelper is the core logic for nuking AWS resources.
// It retrieves resources, confirms deletion with the user, and executes the nuke operation.
func awsNukeHelper(c *cli.Context, configObj config.Config, query *aws.Query, outputFormat string, outputFile string) error {
	// Setup reporting - cleanup calls Complete() and closes writer
	collector, cleanup, err := setupAwsReporting(outputFormat, outputFile, query)
	if err != nil {
		return err
	}
	defer cleanup()

	// Emit scan started event with query parameters
	collector.Emit(buildAwsScanStarted(query))

	// Retrieve all matching resources (emits ResourceFound events via collector)
	account, err := aws.GetAllResources(c.Context, query, configObj, collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting resources",
		}, map[string]interface{}{})
		return errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
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
		return aws.NukeAllResources(account, query.Regions, collector)
	}

	return nil
}

// generateQuery builds an AWS Query object from CLI context and flags.
// The query determines which AWS resources will be targeted for inspection or deletion.
func generateQuery(c *cli.Context, includeUnaliasedKmsKeys bool, overridingResourceTypes []string, onlyDefault bool) (*aws.Query, error) {
	// Parse time-based filters
	excludeAfter, err := parseDurationParam(FlagOlderThan, c.String(FlagOlderThan))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	includeAfter, err := parseDurationParam(FlagNewerThan, c.String(FlagNewerThan))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// Parse timeout
	timeout, err := parseTimeoutDurationParam(FlagTimeout, c.String(FlagTimeout))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// Determine which resource types to target
	resourceTypes := c.StringSlice(FlagResourceType)
	if overridingResourceTypes != nil {
		resourceTypes = overridingResourceTypes
	}

	// Build and return the query
	return aws.NewQuery(
		c.StringSlice(FlagRegion),
		c.StringSlice(FlagExcludeRegion),
		resourceTypes,
		c.StringSlice(FlagExcludeResourceType),
		excludeAfter,
		includeAfter,
		includeUnaliasedKmsKeys,
		timeout,
		onlyDefault,
		c.Bool(FlagExcludeFirstSeen),
	)
}

// handleGetResourcesWithFormat retrieves all AWS resources matching the query and renders them
// in the specified output format. This is used for inspect operations only.
func handleGetResourcesWithFormat(c *cli.Context, configObj config.Config, query *aws.Query, outputFormat string, outputFile string) (
	*aws.AwsAccountResources, error) {
	// Setup reporting - cleanup calls Complete() and closes writer
	collector, cleanup, err := setupAwsReporting(outputFormat, outputFile, query)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Emit scan started event with query parameters
	collector.Emit(buildAwsScanStarted(query))

	// Retrieve all resources matching the query (emits ResourceFound events via collector)
	accountResources, err := aws.GetAllResources(c.Context, query, configObj, collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return nil, errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
	}

	// Signal scan complete - renderer will show found resources table
	collector.Emit(reporting.ScanComplete{})

	return accountResources, nil
}

// handleListResourceTypes displays all available AWS resource types that can be targeted.
func handleListResourceTypes() error {
	return printResourceTypes("AWS Resource Types", aws.ListResourceTypes())
}

// setupAwsReporting creates a collector and appropriate renderer for AWS operations.
// Returns the collector, cleanup function (which calls Complete() and closes writer), and any error.
func setupAwsReporting(outputFormat string, outputFile string, query *aws.Query) (
	*reporting.Collector, func(), error) {
	// Build query params for JSON output
	queryParams := &renderers.QueryParams{
		Regions:              query.Regions,
		ResourceTypes:        query.ResourceTypes,
		ListUnaliasedKMSKeys: query.ListUnaliasedKMSKeys,
	}
	if query.ExcludeAfter != nil && !query.ExcludeAfter.IsZero() {
		queryParams.ExcludeAfter = query.ExcludeAfter
	}
	if query.IncludeAfter != nil && !query.IncludeAfter.IsZero() {
		queryParams.IncludeAfter = query.IncludeAfter
	}

	return setupReporting(outputFormat, outputFile, renderers.JSONRendererConfig{
		Command: "aws",
		Query:   queryParams,
		Regions: query.Regions,
	})
}

// buildAwsScanStarted creates a ScanStarted event from an AWS query.
func buildAwsScanStarted(query *aws.Query) reporting.ScanStarted {
	event := reporting.ScanStarted{
		Regions:              query.Regions,
		ResourceTypes:        query.ResourceTypes,
		ListUnaliasedKMSKeys: query.ListUnaliasedKMSKeys,
	}
	if query.ExcludeAfter != nil && !query.ExcludeAfter.IsZero() {
		event.ExcludeAfter = query.ExcludeAfter.Format("2006-01-02 15:04:05")
	}
	if query.IncludeAfter != nil && !query.IncludeAfter.IsZero() {
		event.IncludeAfter = query.IncludeAfter.Format("2006-01-02 15:04:05")
	}
	return event
}
