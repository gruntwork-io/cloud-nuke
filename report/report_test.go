package report

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"
)

func TestRecordSingleEntry(t *testing.T) {
	e := Entry{
		Identifier:   "arn:aws:sns:us-east-1:999999999999:TestTopic",
		ResourceType: "SNS Topic",
		Error:        nil,
	}
	Record(e)
	testPrintContains(t, e.Identifier)
	testPrintContains(t, "✅")
}

func TestRecordSingleEntryErrorState(t *testing.T) {
	e := Entry{
		Identifier:   "arn:aws:sns:us-east-1:999999999999:TestTopic",
		ResourceType: "SNS Topic",
		Error:        errors.New("This place is not a place of honor..."),
	}
	Record(e)
	testPrintContains(t, e.Identifier)
	testPrintContains(t, "❌")
}

func TestRecordBatchEntries(t *testing.T) {
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
	testPrintContains(t, ids[0])
	testPrintContains(t, ids[1])
	testPrintContains(t, "✅")
}

func TestRecordBatchEntriesErrorState(t *testing.T) {
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
	testPrintContains(t, ids[0])
	testPrintContains(t, ids[1])
}

// Test helpers

// testPrintContains can be used to test Print methods.
func testPrintContains(t *testing.T, match string) {
	output := captureStdout(Print)
	require.True(t, strings.Contains(output, match))
}

var outBuf bytes.Buffer

// setupStdoutCapture sets up a fake stdout capture.
func setupStdoutCapture() {
	outBuf.Reset()
	pterm.SetDefaultOutput(&outBuf)
}

// teardownStdoutCapture restores the real stdout.
func teardownStdoutCapture() {
	pterm.SetDefaultOutput(os.Stdout)
	outBuf.Reset()
}

// captureStdout simulates capturing of os.stdout with a buffer and returns what was writted to the screen
func captureStdout(f func(w io.Writer)) string {
	setupStdoutCapture()
	f(&outBuf)
	return readStdout()
}

// readStdout reads the current stdout buffor. Assumes setupStdoutCapture() has been called before.
func readStdout() string {
	content := outBuf.String()
	outBuf.Reset()
	return content
}
