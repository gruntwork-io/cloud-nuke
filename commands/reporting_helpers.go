package commands

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/urfave/cli/v2"
)

// reportingContext bundles the collector, context, and cleanup function
type reportingContext struct {
	Collector *reporting.Collector
	Ctx       context.Context
	Cleanup   func() error
}

// setupInspectReporting creates a collector with inspect renderer for the given format
func setupInspectReporting(c *cli.Context, outputFormat, outputFile, command string, query renderers.QueryParams) (*reportingContext, error) {
	collector := reporting.NewCollector()

	writer, closer, err := ui.GetOutputWriter(outputFile)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if outputFormat == "json" {
		collector.AddRenderer(renderers.NewInspectJSONRenderer(writer, command, query))
	} else {
		collector.AddRenderer(renderers.NewInspectCLIRenderer(writer))
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = reporting.WithCollector(ctx, collector)

	return &reportingContext{
		Collector: collector,
		Ctx:       ctx,
		Cleanup:   closer,
	}, nil
}

// setupNukeReporting creates a collector with nuke renderer for the given format
func setupNukeReporting(c *cli.Context, outputFormat, outputFile, command string, regions []string) (*reportingContext, error) {
	collector := reporting.NewCollector()

	writer, closer, err := ui.GetOutputWriter(outputFile)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if outputFormat == "json" {
		collector.AddRenderer(renderers.NewNukeJSONRenderer(writer, command, regions))
	} else {
		collector.AddRenderer(renderers.NewNukeCLIRenderer(writer))
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = reporting.WithCollector(ctx, collector)

	return &reportingContext{
		Collector: collector,
		Ctx:       ctx,
		Cleanup:   closer,
	}, nil
}
