package report

import (
	"sync"

	"github.com/gruntwork-io/cloud-nuke/progressbar"
)

var m = &sync.Mutex{}

var records = make(map[string]Entry)

func GetRecords() map[string]Entry {
	return records
}

func ResetRecords() {
	records = make(map[string]Entry)
}

func Record(e Entry) {
	defer m.Unlock()
	m.Lock()
	records[e.Identifier] = e
	// Increment the progressbar so the user feels measurable progress on long-running nuke jobs
	p := progressbar.GetProgressbar()
	// Don't increment the progressbar when running tests
	if p.IsActive {
		p.Increment()
	}
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
