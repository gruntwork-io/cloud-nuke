package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/azure"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
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

	ctx, cancel := withTimeout(c.Context, query.Timeout)
	defer cancel()

	account, err := azure.GetAllResources(ctx, query, configObj, collector)
	if err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{EventName: "Error getting resources"}, map[string]interface{}{})
		return errors.WithStackTrace(timeoutError(err, query.Timeout))
	}
	if err := timedOut(ctx, query.Timeout); err != nil {
		return errors.WithStackTrace(err)
	}

	collector.Emit(reporting.ScanComplete{})

	shouldProceed, err := confirmNuke(c, len(account.Resources) > 0)
	if err != nil {
		return err
	}

	if shouldProceed {
		err := azure.NukeSubscriptionResources(ctx, account, query.SubscriptionIDs, query.Parallelism, collector)
		return timeoutError(err, query.Timeout)
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

	ctx, cancel := withTimeout(c.Context, query.Timeout)
	defer cancel()

	if _, err := azure.GetAllResources(ctx, query, configObj, collector); err != nil {
		telemetry.TrackEvent(commonTelemetry.EventContext{EventName: "Error inspecting resources"}, map[string]interface{}{})
		return errors.WithStackTrace(timeoutError(err, query.Timeout))
	}
	if err := timedOut(ctx, query.Timeout); err != nil {
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

	// --timeout is read here rather than through parseAndApplyTimeout, which
	// writes it into the config where nothing reads it back. The handlers turn
	// it into a deadline on the context instead.
	timeout, err := parseTimeoutDurationParam(FlagTimeout, c.String(FlagTimeout))
	if err != nil {
		return nil, config.Config{}, errors.WithStackTrace(err)
	}
	query.Timeout = timeout
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

// withTimeout turns --timeout into a deadline over the whole run: the scan, the
// prompt and the nuke. Azure has no per-resource timeout to apply it to, and a
// flag the help text advertises should do something.
func withTimeout(ctx context.Context, timeout *time.Duration) (context.Context, context.CancelFunc) {
	if timeout == nil || *timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, *timeout)
}

// timeoutError names the deadline beside whatever else the run reported,
// because "context deadline exceeded" does not say which deadline or how long
// it was, and a run that failed three deletes and then ran out of time has to
// report both.
func timeoutError(err error, timeout *time.Duration) error {
	if err == nil || timeout == nil || !stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %w", util.ResourceExecutionTimeout{Timeout: *timeout}, err)
}

// timedOut reports the deadline a scan passed without failing. The engine
// records a listing failure and carries on, which is right for one resource
// type among many and wrong for a deadline: without this, a run whose every
// list was cut short reports a clean subscription and exits successfully.
func timedOut(ctx context.Context, timeout *time.Duration) error {
	if ctx.Err() == nil || timeout == nil || !stderrors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil
	}
	return util.ResourceExecutionTimeout{Timeout: *timeout}
}
