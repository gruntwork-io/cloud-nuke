package report

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordSingleEntry(t *testing.T) {
	e := Entry{
		Identifier:   "arn:aws:sns:us-east-1:999999999999:TestTopic",
		ResourceType: "SNS Topic",
		Error:        nil,
	}
	Record(e)
	ensureRecordsContainIdentifier(t, e.Identifier)
}

func TestRecordSingleEntryErrorState(t *testing.T) {
	ResetRecords()
	e := Entry{
		Identifier:   "arn:aws:sns:us-east-1:999999999999:TestTopic",
		ResourceType: "SNS Topic",
		Error:        errors.New("This place is not a place of honor..."),
	}
	Record(e)
	entry := getTestRecord(e.Identifier)
	require.NotNil(t, entry)
	require.Equal(t, entry.Error, e.Error)
}

func TestRecordBatchEntries(t *testing.T) {
	ResetRecords()

	ids := []string{
		"arn:aws:sns:us-east-1:999999999999:TestTopic",
		"arn:aws:sns:us-east-1:111111111111:TestTopicTwo",
	}

	be := BatchEntry{
		Identifiers:  ids,
		ResourceType: "SNS Topic",
		Error:        nil,
	}
	RecordBatch(be)
	for _, id := range ids {
		ensureRecordsContainIdentifier(t, id)
	}
}

func TestRecordBatchEntriesErrorState(t *testing.T) {
	ResetRecords()

	ids := []string{
		"arn:aws:sns:us-east-1:999999999999:TestTopic",
		"arn:aws:sns:us-east-1:111111111111:TestTopicTwo",
	}

	be := BatchEntry{
		Identifiers:  ids,
		ResourceType: "SNS Topic",
		Error:        errors.New("no highly esteemed deed is commemorated here...nothing valued is here."),
	}
	RecordBatch(be)
	for _, id := range ids {
		ensureRecordsContainIdentifier(t, id)
	}
	entry1 := getTestRecord(ids[0])
	require.NotNil(t, entry1)
	require.Equal(t, entry1.Error, be.Error)

	entry2 := getTestRecord(ids[1])
	require.NotNil(t, entry2)
	require.Equal(t, entry2.Error, be.Error)
}

// Test helpers

func ensureRecordsContainIdentifier(t *testing.T, key string) {
	records := GetRecords()
	found := false
	for k := range records {
		if k == key {
			found = true
		}
	}
	if found == false {
		t.Fail()
	}
}

func getTestRecord(key string) *Entry {
	records := GetRecords()
	for k, entry := range records {
		if k == key {
			return &entry
		}
	}
	return nil
}
