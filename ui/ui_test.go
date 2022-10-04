package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"
)

// testPrintContains can be used to test Print methods.
func testPrintContains(t *testing.T, match string) {
	output := captureStdout(PrintRunReport)
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
