package renderers

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONRenderer_InspectOutput(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "inspect-aws",
		Query: &QueryParams{
			Regions:       []string{"us-east-1"},
			ResourceTypes: []string{"ec2"},
		},
	})

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
	// Complete triggers JSON output
	r.OnEvent(reporting.Complete{})

	var output InspectOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "inspect-aws", output.Command)
	assert.Len(t, output.Resources, 2)
	assert.Len(t, output.Errors, 1)
	assert.Equal(t, 2, output.Summary.TotalResources)
	assert.Equal(t, 1, output.Summary.Nukable)
	assert.Equal(t, 1, output.Summary.NonNukable)
	assert.Equal(t, 1, output.Summary.GeneralErrors)
}

func TestJSONRenderer_NukeOutput(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "aws",
		Regions: []string{"us-east-1", "us-west-2"},
	})

	// ResourceFound events (scan phase)
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
		Nukable:      true,
	})
	r.OnEvent(reporting.ResourceFound{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-789",
		Nukable:      false,
		Reason:       "protected",
	})

	// Simulate nuke flow: ScanComplete, then NukeStarted, deletions, NukeComplete
	r.OnEvent(reporting.ScanComplete{})
	r.OnEvent(reporting.NukeStarted{Total: 2})

	r.OnEvent(reporting.ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-123",
		Success:      true,
	})
	r.OnEvent(reporting.ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-456",
		Success:      false,
		Error:        "access denied",
	})
	r.OnEvent(reporting.GeneralError{
		ResourceType: "s3",
		Description:  "Failed to list",
		Error:        "timeout",
	})

	r.OnEvent(reporting.NukeComplete{})
	// Complete triggers JSON output
	r.OnEvent(reporting.Complete{})

	var output NukeOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "aws", output.Command)
	assert.Len(t, output.Found, 3)
	assert.Len(t, output.Resources, 2)
	assert.Equal(t, 1, output.Summary.Deleted)
	assert.Equal(t, 1, output.Summary.Failed)
}

func TestJSONRenderer_EmptyOutput(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "inspect-aws",
	})

	r.OnEvent(reporting.ScanComplete{})
	// Complete triggers output even with no data
	r.OnEvent(reporting.Complete{})

	var output InspectOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, 0, output.Summary.TotalResources)
	assert.Len(t, output.Resources, 0)
}

// TestJSONRenderer_NukeDoesNotOutputOnScanComplete verifies that nuke commands
// only output on Complete, not on ScanComplete. This prevents writing two
// JSON documents to the same file which would corrupt the output.
func TestJSONRenderer_NukeDoesNotOutputOnScanComplete(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "aws",
		Regions: []string{"us-east-1"},
	})

	// Simulate scan phase
	r.OnEvent(reporting.ResourceFound{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-123",
		Nukable:      true,
	})

	// ScanComplete should NOT trigger output
	r.OnEvent(reporting.ScanComplete{})
	assert.Empty(t, buf.String(), "should not output on ScanComplete")

	// Simulate nuke phase
	r.OnEvent(reporting.NukeStarted{Total: 1})
	r.OnEvent(reporting.ResourceDeleted{
		ResourceType: "ec2",
		Region:       "us-east-1",
		Identifier:   "i-123",
		Success:      true,
	})

	// NukeComplete should NOT trigger output either
	r.OnEvent(reporting.NukeComplete{})
	assert.Empty(t, buf.String(), "should not output on NukeComplete")

	// Only Complete should trigger output
	r.OnEvent(reporting.Complete{})
	assert.NotEmpty(t, buf.String(), "should output on Complete")

	// Verify it's valid JSON with nuke output structure (single document)
	var output NukeOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err, "output should be valid JSON (single document)")
	assert.Equal(t, 1, output.Summary.Deleted)
}
