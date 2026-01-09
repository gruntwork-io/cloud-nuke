// Package renderers provides output renderers for cloud-nuke operations.
//
// Renderers receive events from a Collector and produce output in various formats.
// Available renderers:
//   - CLIRenderer: Table output for terminal
//   - JSONRenderer: JSON output for programmatic consumption
package renderers

import "github.com/gruntwork-io/cloud-nuke/reporting"

// Ensure renderers implement the Renderer interface
var (
	_ reporting.Renderer = (*CLIRenderer)(nil)
	_ reporting.Renderer = (*JSONRenderer)(nil)
)
