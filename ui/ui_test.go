package ui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"
)

func TestRenderEntriesWithNoErrors(t *testing.T) {
	report.ResetRecords()

	e := report.Entry{
		Identifier:   "arn:aws:sns:us-west-1:222222222222:DifferentTopic",
		ResourceType: "SNS Topic",
		Error:        nil,
	}
	report.Record(e)

	ensureRenderedReportContains(t, e.Identifier)
	ensureRenderedReportContains(t, SuccessEmoji)
	ensureRenderedReportDoesNotContain(t, FailureEmoji)
}

func TestRenderEntriesWithErrors(t *testing.T) {
	report.ResetRecords()

	e := report.Entry{
		Identifier:   "arn:aws:sns:ap-southeast-1:222222222222:DifferentTopic",
		ResourceType: "SNS Topic",
		Error:        errors.New("What is here was dangerous and repulsive to us. This message is a warning about danger. "),
	}
	report.Record(e)

	ensureRenderedReportContains(t, e.Identifier)
	ensureRenderedReportContains(t, FailureEmoji)
	ensureRenderedReportDoesNotContain(t, SuccessEmoji)
}

// testPrintContains can be used to test Print methods.
func ensureRenderedReportContains(t *testing.T, match string) {
	output := captureStdout(PrintRunReport)
	require.True(t, strings.Contains(output, match))
}

func ensureRenderedReportDoesNotContain(t *testing.T, match string) {
	output := captureStdout(PrintRunReport)
	require.False(t, strings.Contains(output, match))
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
