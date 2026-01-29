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

	// ScanComplete triggers JSON output for inspect mode
	r.OnEvent(reporting.ScanComplete{})

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

	// NukeStarted sets nuke mode
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

	// NukeComplete triggers JSON output
	r.OnEvent(reporting.NukeComplete{})

	var output NukeOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "aws", output.Command)
	assert.Equal(t, []string{"us-east-1", "us-west-2"}, output.Regions)
	assert.Len(t, output.Found, 3)
	assert.Len(t, output.Resources, 2)
	assert.Len(t, output.Errors, 1)
	assert.Equal(t, 3, output.Summary.Found)
	assert.Equal(t, 2, output.Summary.Total)
	assert.Equal(t, 1, output.Summary.Deleted)
	assert.Equal(t, 1, output.Summary.Failed)
	assert.Equal(t, 1, output.Summary.GeneralErrors)

	// Check found resources
	assert.Equal(t, "i-123", output.Found[0].Identifier)
	assert.True(t, output.Found[0].Nukable)
	assert.Equal(t, "i-789", output.Found[2].Identifier)
	assert.False(t, output.Found[2].Nukable)
	assert.Equal(t, "protected", output.Found[2].Reason)

	// Check resource statuses
	assert.Equal(t, "deleted", output.Resources[0].Status)
	assert.Equal(t, "failed", output.Resources[1].Status)
	assert.Equal(t, "access denied", output.Resources[1].Error)
}

func TestJSONRenderer_EmptyOutput(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "inspect-aws",
	})

	// ScanComplete triggers output even with no data
	r.OnEvent(reporting.ScanComplete{})

	var output InspectOutput
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, 0, output.Summary.TotalResources)
	assert.Len(t, output.Resources, 0)
}
