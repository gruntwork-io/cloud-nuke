package report

import (
	"sync"

	"github.com/andrewderr/cloud-nuke-a1/progressbar"
)

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
	// Increment the progressbar so the user feels measurable progress on long-running nuke jobs
	p := progressbar.GetProgressbar()
	p.Increment()
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

// Custom types
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
