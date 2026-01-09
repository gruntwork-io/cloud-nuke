package renderers

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONRenderer_Nuke(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "aws",
		Regions: []string{"us-east-1", "us-west-2"},
	})

	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Identifier: "i-1", Success: true})
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Identifier: "i-2", Success: false, Error: "denied"})
	r.OnEvent(reporting.GeneralError{ResourceType: "s3", Description: "list failed", Error: "timeout"})
	r.OnEvent(reporting.ResourceFound{}) // collected but not rendered for nuke

	assert.Len(t, r.deleted, 2)
	assert.Len(t, r.errors, 1)

	require.NoError(t, r.Render())

	var output NukeOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	assert.Equal(t, "aws", output.Command)
	assert.Len(t, output.Resources, 2)
	assert.Len(t, output.Errors, 1)
	assert.Equal(t, 1, output.Summary.Deleted)
	assert.Equal(t, 1, output.Summary.Failed)
}

func TestJSONRenderer_Inspect(t *testing.T) {
	var buf bytes.Buffer
	query := QueryParams{Regions: []string{"us-east-1"}, ResourceTypes: []string{"ec2"}}
	r := NewJSONRenderer(&buf, JSONRendererConfig{
		Command: "inspect-aws",
		Query:   &query,
	})

	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-1", Nukable: true})
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-2", Nukable: false, Reason: "protected"})
	// Note: With unified renderer, output type is determined by collected events
	// Only ResourceFound events → InspectOutput

	assert.Len(t, r.found, 2)

	require.NoError(t, r.Render())

	var output InspectOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	assert.Equal(t, "inspect-aws", output.Command)
	assert.Len(t, output.Resources, 2)
	assert.Equal(t, 1, output.Summary.Nukable)
	assert.Equal(t, 1, output.Summary.NonNukable)
}

func TestJSONRenderer_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf, JSONRendererConfig{Command: "inspect-aws"})

	require.NoError(t, r.Render())

	var output InspectOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))
	assert.Equal(t, 0, output.Summary.TotalResources)
}
