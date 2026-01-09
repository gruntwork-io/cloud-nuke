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

// ReportingConfig holds configuration for setting up reporting
type ReportingConfig struct {
	OutputFormat string
	OutputFile   string
	Command      string
	Query        *renderers.QueryParams // for inspect operations
	Regions      []string               // for nuke operations
}

// setupReporting creates a collector with appropriate renderer
func setupReporting(c *cli.Context, cfg ReportingConfig) (*reportingContext, error) {
	collector := reporting.NewCollector()

	writer, closer, err := ui.GetOutputWriter(cfg.OutputFile)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if cfg.OutputFormat == "json" {
		collector.AddRenderer(renderers.NewJSONRenderer(writer, renderers.JSONRendererConfig{
			Command: cfg.Command,
			Query:   cfg.Query,
			Regions: cfg.Regions,
		}))
	} else {
		collector.AddRenderer(renderers.NewCLIRenderer(writer))
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return &reportingContext{
		Collector: collector,
		Ctx:       ctx,
		Cleanup:   closer,
	}, nil
}
