package commands

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
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

	if c.Bool(FlagListResourceTypes) {
		return handleListResourceTypes()
	}

	if err := parseLogLevel(c); err != nil {
		return err
	}

	query, err := generateQuery(c, c.Bool(FlagListUnaliasedKMSKeys), nil, false)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	outputFormat := c.String(FlagOutputFormat)
	outputFile := c.String(FlagOutputFile)

	rc, err := setupReporting(c, ReportingConfig{
		OutputFormat: outputFormat,
		OutputFile:   outputFile,
		Command:      "inspect-aws",
		Query: &renderers.QueryParams{
			Regions:              query.Regions,
			ResourceTypes:        query.ResourceTypes,
			ExcludeAfter:         query.ExcludeAfter,
			IncludeAfter:         query.IncludeAfter,
			ListUnaliasedKMSKeys: query.ListUnaliasedKMSKeys,
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = rc.Cleanup() }()

	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("AWS Resource Query Parameters")
		if err := ui.RenderQueryAsBulletList(query); err != nil {
			return errors.WithStackTrace(err)
		}
		pterm.Println()
	}

	_, err = aws.GetAllResources(rc.Ctx, query, config.Config{}, rc.Collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error inspecting resources",
		}, map[string]interface{}{})
		_ = rc.Collector.Complete()
		return errors.WithStackTrace(aws.ResourceInspectionError{Underlying: err})
	}

	if !ui.ShouldSuppressProgressOutput(outputFormat) {
		pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println("Found AWS Resources")
	}

	return rc.Collector.Complete()
}

// Helper Functions
// These functions contain shared logic used by multiple command handlers

// awsNukeHelper is the core logic for nuking AWS resources.
// It retrieves resources, confirms deletion with the user, and executes the nuke operation.
func awsNukeHelper(c *cli.Context, configObj config.Config, query *aws.Query, outputFormat string, outputFile string) error {
	rc, err := setupReporting(c, ReportingConfig{
		OutputFormat: outputFormat,
		OutputFile:   outputFile,
		Command:      "aws",
		Regions:      query.Regions,
	})
	if err != nil {
		return err
	}
	defer func() { _ = rc.Cleanup() }()

	account, err := handleGetResourcesWithFormatCtx(rc.Ctx, configObj, query, outputFormat, outputFile, rc.Collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error getting resources",
		}, map[string]interface{}{})
		_ = rc.Collector.Complete()
		return errors.WithStackTrace(err)
	}

	shouldProceed, err := confirmNuke(c, len(account.Resources) > 0)
	if err != nil {
		return err
	}

	if shouldProceed {
		if err := aws.NukeAllResources(rc.Ctx, account, query.Regions, rc.Collector); err != nil {
			_ = rc.Collector.Complete()
			return err
		}
		if err := rc.Collector.Complete(); err != nil {
			return err
		}
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

// handleGetResourcesWithFormatCtx retrieves all AWS resources matching the query and renders them
// in the specified output format. Accepts an optional collector to capture errors.
// This is used for nuke operations where we want to capture errors in the collector.
func handleGetResourcesWithFormatCtx(ctx context.Context, configObj config.Config, query *aws.Query, outputFormat string, outputFile string, collector *reporting.Collector) (
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
	accountResources, err := aws.GetAllResources(ctx, query, configObj, collector)
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
