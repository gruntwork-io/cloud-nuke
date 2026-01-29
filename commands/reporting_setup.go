package commands

import (
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/renderers"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/pterm/pterm"
)

// setupReporting creates a collector and appropriate renderer based on output format.
// Returns the collector, cleanup function (which calls Complete() and closes writer), and any error.
// The jsonConfig is used when outputFormat is "json"; ignored otherwise.
func setupReporting(outputFormat string, outputFile string, jsonConfig renderers.JSONRendererConfig) (
	*reporting.Collector, func(), error) {
	writer, writerCleanup, err := renderers.GetOutputWriter(outputFile)
	if err != nil {
		return nil, nil, err
	}

	collector := reporting.NewCollector()

	// Combined cleanup: mark collector closed then close writer
	cleanup := func() {
		collector.Complete()
		if err := writerCleanup(); err != nil {
			logging.Errorf("Failed to close output writer: %v", err)
		}
	}

	if outputFormat == "json" {
		collector.AddRenderer(renderers.NewJSONRenderer(writer, jsonConfig))
		return collector, cleanup, nil
	}

	// CLI format
	collector.AddRenderer(renderers.NewCLIRenderer(writer))
	return collector, cleanup, nil
}

// printResourceTypes prints a list of resource types with a section header.
// This is a simple helper that doesn't use the event-driven pattern.
func printResourceTypes(sectionTitle string, resourceTypes []string) error {
	pterm.DefaultSection.WithTopPadding(1).WithBottomPadding(0).Println(sectionTitle)

	var items []pterm.BulletListItem
	for _, resourceType := range resourceTypes {
		items = append(items, pterm.BulletListItem{Level: 0, Text: resourceType})
	}

	return pterm.DefaultBulletList.WithItems(items).Render()
}
