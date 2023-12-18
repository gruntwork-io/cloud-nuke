package report

import (
	"sync"
)

/**
Using a global variables to keep track of reports & errors.

Generally, it is a bad idea to use global variables. However, in this case, it is the easiest way to keep track of
data that are running in parallel. We can explore passing reports/errors from individual resources in the future.
*/

// Global variables to keep track of errors/progress when nuking resources.
var m = &sync.Mutex{}
var generalErrors = make(map[string]GeneralError)
var records = make(map[string]Entry)

func GetRecords() map[string]Entry {
	return records
}

func GetErrors() map[string]GeneralError {
	return generalErrors
}

func ResetRecords() {
	records = make(map[string]Entry)
}

func ResetErrors() {
	generalErrors = make(map[string]GeneralError)
}

func Record(e Entry) {
	defer m.Unlock()
	m.Lock()
	records[e.Identifier] = e
}

// RecordBatch accepts a BatchEntry that contains a slice of identifiers, loops through them and converts each identifier to
// a standard Entry. This is useful for supporting batch delete workflows in cloud-nuke (such as cloudwatch_dashboards)
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
