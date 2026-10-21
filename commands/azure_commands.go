package commands

import (
	"github.com/gruntwork-io/cloud-nuke/azure"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/urfave/cli/v2"
)

// Azure Command Handlers
// These functions implement the CLI commands for Azure operations

// azureNuke is the main command handler for nuking (deleting) Azure resources.
func azureNuke(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("azure")()

	if c.Bool(FlagListResourceTypes) {
		return handleListAzureResourceTypes()
	}

	query, configObj, err := azureQueryFrom(c)
	if err != nil {
		return err
	}

	collector, cleanup, err := setupAzureReporting(c.String(FlagOutputFormat), c.String(FlagOutputFile), query.SubscriptionIDs)
	if err != nil {
		return err
	}
	defer cleanup()

	account, err := azure.GetAllResources(c.Context, query, configObj, collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{EventName: "Error getting resources"}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	collector.Emit(reporting.ScanComplete{})

	shouldProceed, err := confirmNuke(c, len(account.Resources) > 0)
	if err != nil {
		return err
	}

	if shouldProceed {
		return azure.NukeSubscriptionResources(c.Context, account, query.SubscriptionIDs, query.Parallelism, collector)
	}

	return nil
}

// azureInspect lists the Azure resources that would be deleted, deleting none.
func azureInspect(c *cli.Context) error {
	defer telemetry.TrackCommandLifecycle("azure-inspect")()

	if c.Bool(FlagListResourceTypes) {
		return handleListAzureResourceTypes()
	}

	query, configObj, err := azureQueryFrom(c)
	if err != nil {
		return err
	}

	collector, cleanup, err := setupAzureReporting(c.String(FlagOutputFormat), c.String(FlagOutputFile), query.SubscriptionIDs)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := azure.GetAllResources(c.Context, query, configObj, collector); err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{EventName: "Error inspecting resources"}, map[string]interface{}{})
		return errors.WithStackTrace(err)
	}

	collector.Emit(reporting.ScanComplete{})
	return nil
}

// azureQueryFrom builds and validates the query both commands share, so the
// two cannot drift apart the way the AWS and GCP handlers did.
func azureQueryFrom(c *cli.Context) (*azure.Query, config.Config, error) {
	if err := parseLogLevel(c); err != nil {
		return nil, config.Config{}, err
	}

	configObj, err := loadConfigFile(c.String(FlagConfig))
	if err != nil {
		return nil, config.Config{}, errors.WithStackTrace(err)
	}

	query := &azure.Query{
		SubscriptionIDs:      c.StringSlice(FlagSubscriptionID),
		Locations:            c.StringSlice(FlagRegion),
		ExcludeLocations:     c.StringSlice(FlagExcludeRegion),
		ResourceTypes:        c.StringSlice(FlagResourceType),
		ExcludeResourceTypes: c.StringSlice(FlagExcludeResourceType),
		Parallelism:          c.Int(FlagParallelism),
	}

	if err := parseAndApplyTimeout(c, &configObj); err != nil {
		return nil, config.Config{}, err
	}
	if err := parseAndApplyTimeFilters(c, &configObj); err != nil {
		return nil, config.Config{}, err
	}

	if err := query.Validate(); err != nil {
		return nil, config.Config{}, err
	}

	return query, configObj, nil
}

// handleListAzureResourceTypes displays every Azure resource type.
func handleListAzureResourceTypes() error {
	return printResourceTypes("Azure Resource Types", azure.ListResourceTypes())
}

// setupAzureReporting creates a collector and renderer for Azure operations.
func setupAzureReporting(outputFormat string, outputFile string, subscriptionIDs []string) (*reporting.Collector, func(), error) {
	return setupReporting(outputFormat, outputFile, renderers.JSONRendererConfig{
		Command: "azure",
		Regions: subscriptionIDs,
	})
}
