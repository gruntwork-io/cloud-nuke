package commands

import (
	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

// AWS Command Handlers
// These functions implement the CLI commands for AWS operations

// awsNuke is the main command handler for nuking (deleting) AWS resources.
// It supports resource filtering, time-based filtering, and config file overrides.
func awsNuke(c *cli.Context) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws",
	}, map[string]interface{}{})

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
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws-defaults",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws-defaults",
	}, map[string]interface{}{})

	// Parse and set log level
	parseErr := logging.ParseLogLevel(c.String(FlagLogLevel))
	if parseErr != nil {
		return errors.WithStackTrace(parseErr)
	}

	// Determine which default resources to target based on flags
	resourceTypes := []string{"vpc"} // VPC deletion will remove attached default security groups
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
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Start aws-inspect",
	}, map[string]interface{}{})
	defer telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "End aws-inspect",
	}, map[string]interface{}{})

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
	// Retrieve all matching resources
	account, err := handleGetResourcesWithFormat(c, configObj, query, outputFormat, outputFile)
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
		if err := aws.NukeAllResources(account, query.Regions); err != nil {
			return err
		}
		ui.RenderRunReportWithFormat(outputFormat, outputFile)
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
// in the specified output format. This is used for both inspect and nuke operations.
func handleGetResourcesWithFormat(c *cli.Context, configObj config.Config, query *aws.Query, outputFormat string, outputFile string) (
	*aws.AwsAccountResources, error) {
	// Display query parameters (only for table format to avoid cluttering JSON output)
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("AWS Resource Query Parameters")
		err := ui.RenderQueryAsBulletList(query)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		pterm.Println()
	}

	// Retrieve all resources matching the query
	accountResources, err := aws.GetAllResources(c.Context, query, configObj)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		return nil, errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
	}

	// Display found resources header (only for table format)
	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Found AWS Resources")
	}

	// Render the resources in the requested format (table or JSON)
	err = ui.RenderResourcesAsTableWithFormat(accountResources, query, outputFormat, outputFile)

	return accountResources, err
}

// handleListResourceTypes displays all available AWS resource types that can be targeted.
func handleListResourceTypes() error {
	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("AWS Resource Types")
	return ui.RenderResourceTypesAsBulletList(aws.ListResourceTypes())
}
