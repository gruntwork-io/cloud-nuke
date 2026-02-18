package renderers

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/stretchr/testify/assert"
)

func TestCLIRenderer_OnEvent(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// Scan phase events
	r.OnEvent(reporting.ResourceFound{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-123",
		Nukable:      true,
	})
	r.OnEvent(reporting.ResourceFound{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-456",
		Nukable:      false,
		Reason:       "protected",
	})
	r.OnEvent(reporting.GeneralError{
		ResourceType: "s3",
		Description:  "Failed to list",
		Error:        "timeout",
	})
	r.OnEvent(reporting.ScanComplete{})

	// Nuke phase events
	r.OnEvent(reporting.NukeStarted{Total: 1})
	r.OnEvent(reporting.ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-123",
		Success:      true,
	})
	r.OnEvent(reporting.NukeComplete{})

	assert.Len(t, r.found, 2)
	assert.Len(t, r.deleted, 1)
	assert.Len(t, r.errors, 1)
}

func TestCLIRenderer_EmptyRender(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// ScanComplete with no data should print "No resources found"
	r.OnEvent(reporting.ScanComplete{})

	output := buf.String()
	assert.Contains(t, output, "No resources found")
}

func TestCLIRenderer_ProgressEvents(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// Spinner is active initially
	assert.NotNil(t, r.spinner)
	assert.Nil(t, r.progressBar)
	assert.False(t, r.nukeMode)

	// ScanComplete stops spinner (this happens before NukeStarted in real flow)
	r.OnEvent(reporting.ScanComplete{})
	assert.Nil(t, r.spinner, "spinner should be stopped after ScanComplete")

	// NukeStarted starts progress bar and sets nukeMode
	r.OnEvent(reporting.NukeStarted{Total: 10})
	assert.NotNil(t, r.progressBar, "progress bar should be started after NukeStarted")
	assert.True(t, r.nukeMode, "nukeMode should be true after NukeStarted")

	// NukeProgress updates title
	r.OnEvent(reporting.NukeProgress{
		ResourceType: "ec2",
		Region:       "us-east-1",
		BatchSize:    5,
	})

	// ResourceDeleted increments counter and collects event
	r.OnEvent(reporting.ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-123",
		Success:      true,
	})
	assert.Len(t, r.deleted, 1)

	// NukeComplete stops progress bar
	r.OnEvent(reporting.NukeComplete{})
	assert.Nil(t, r.progressBar, "progress bar should be stopped after NukeComplete")
}

func TestCLIRenderer_LargeDatasetSummary(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// Emit more than MaxResourcesForDetailedTable ResourceFound events
	count := MaxResourcesForDetailedTable + 100
	for i := 0; i < count; i++ {
		r.OnEvent(reporting.ResourceFound{
			ResourceType: "sns-topic",
			Region:       "us-east-1",
			Identifier:   fmt.Sprintf("arn:aws:sns:us-east-1:123456789012:topic-%d", i),
			Nukable:      true,
		})
	}
	r.OnEvent(reporting.ScanComplete{})

	output := buf.String()

	// Should contain the summary info message
	assert.Contains(t, output, "summary")
	assert.Contains(t, output, fmt.Sprintf("%d resources total", count))
	assert.Contains(t, output, "--output json")

	// Should NOT contain individual resource identifiers
	assert.NotContains(t, output, "topic-0")
	assert.NotContains(t, output, "topic-100")
}

func TestCLIRenderer_LargeDeletedDatasetSummary(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// Set up nuke mode
	count := MaxResourcesForDetailedTable + 50
	r.OnEvent(reporting.NukeStarted{Total: count})

	for i := 0; i < count; i++ {
		r.OnEvent(reporting.ResourceDeleted{
			ResourceType: "sns-topic",
			Region:       "us-east-1",
			Identifier:   fmt.Sprintf("arn:aws:sns:us-east-1:123456789012:topic-%d", i),
			Success:      true,
		})
	}
	r.OnEvent(reporting.NukeComplete{})

	output := buf.String()

	assert.Contains(t, output, "summary")
	assert.Contains(t, output, fmt.Sprintf("%d resources deleted", count))
	assert.Contains(t, output, "--output json")
	assert.NotContains(t, output, "topic-0")
}
