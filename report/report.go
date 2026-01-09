// Package report provides legacy global-state based reporting for cloud-nuke operations.
//
// Deprecated: This package is deprecated and maintained for backward compatibility only.
// New code should use the reporting package with the Collector/Renderer pattern instead.
// The reporting package provides:
//   - Event-driven reporting via reporting.Collector
//   - Multiple output formats via renderers (CLI, JSON)
//   - Thread-safe event collection without global state
//   - Explicit parameter passing for collectors
//
// Migration: Replace report.Record() calls with collector.RecordDeleted() and
// report.RecordError() with collector.RecordError(). Pass the collector explicitly
// as a function parameter to the functions that need it.
package report

import (
	"sync"

	"github.com/gruntwork-io/cloud-nuke/util"
)

/*
Using global variables to keep track of reports & errors.

Generally, it is a bad idea to use global variables. However, in this case, it is the easiest way to keep track of
data that are running in parallel. We can explore passing reports/errors from individual resources in the future.

Deprecated: Use the reporting package with Collector/Renderer pattern instead.
*/

// Global variables to keep track of errors/progress when nuking resources.
var m = &sync.Mutex{}
var generalErrors = make(map[string]GeneralError)
var records = make(map[string]Entry)

// GetRecords returns all recorded entries.
// Deprecated: Use reporting.Collector with appropriate Renderer instead.
func GetRecords() map[string]Entry {
	return records
}

// GetErrors returns all recorded general errors.
// Deprecated: Use reporting.Collector with appropriate Renderer instead.
func GetErrors() map[string]GeneralError {
	return generalErrors
}

// ResetRecords clears all recorded entries.
// Deprecated: Use reporting.Collector with appropriate Renderer instead.
func ResetRecords() {
	records = make(map[string]Entry)
}

// ResetErrors clears all recorded general errors.
// Deprecated: Use reporting.Collector with appropriate Renderer instead.
func ResetErrors() {
	generalErrors = make(map[string]GeneralError)
}

// Record records a deletion result entry.
// Deprecated: Use collector.RecordDeleted() from the reporting package instead.
func Record(e Entry) {
	defer m.Unlock()
	m.Lock()

	// Transform the aws error into custom error format.
	e.Error = util.TransformAWSError(e.Error)
	records[e.Identifier] = e
}

// RecordBatch accepts a BatchEntry that contains a slice of identifiers, loops through them and converts each identifier to
// a standard Entry. This is useful for supporting batch delete workflows in cloud-nuke (such as cloudwatch_dashboards)
// Deprecated: Use collector.RecordDeleted() from the reporting package instead, called for each identifier.
func RecordBatch(e BatchEntry) {
	for _, identifier := range e.Identifiers {
		entry := Entry{
			Identifier:   identifier,
			ResourceType: e.ResourceType,
			Error:        e.Error,
		}
		Record(entry)
	}
}

// RecordError records a general error that occurred during execution.
// Deprecated: Use collector.RecordError() from the reporting package instead.
func RecordError(e GeneralError) {
	defer m.Unlock()
	m.Lock()
	generalErrors[e.Description] = e
}

type Entry struct {
	Identifier   string
	ResourceType string
	Error        error
}

type BatchEntry struct {
	Identifiers  []string
	ResourceType string
	Error        error
}

type GeneralError struct {
	Error        error
	ResourceType string
	Description  string
}
