// Package renderers provides output renderers for cloud-nuke operations.
//
// Renderers receive events from a Collector and produce output in various formats.
// Available renderers:
//   - NukeCLIRenderer: Progress bar + table for nuke operations
//   - InspectCLIRenderer: Table output for inspect operations
//   - NukeJSONRenderer: JSON output for nuke operations
//   - InspectJSONRenderer: JSON output for inspect operations
package renderers

import "github.com/gruntwork-io/cloud-nuke/reporting"

// Ensure renderers implement the Renderer interface
var (
	_ reporting.Renderer = (*NukeCLIRenderer)(nil)
	_ reporting.Renderer = (*InspectCLIRenderer)(nil)
	_ reporting.Renderer = (*NukeJSONRenderer)(nil)
	_ reporting.Renderer = (*InspectJSONRenderer)(nil)
)
