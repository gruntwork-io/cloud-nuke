package renderers

import (
	"bytes"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/stretchr/testify/assert"
)

func TestNukeCLIRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := NewNukeCLIRenderer(&buf)

	// Send events
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Identifier: "i-1", Success: true})
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Identifier: "i-2", Success: false, Error: "denied"})
	r.OnEvent(reporting.GeneralError{ResourceType: "s3", Description: "list failed", Error: "timeout"})
	r.OnEvent(reporting.ResourceFound{}) // should be ignored

	assert.Len(t, r.resources, 2)
	assert.Len(t, r.errors, 1)

	// Render
	assert.NoError(t, r.Render())
	assert.Contains(t, buf.String(), "i-1")
	assert.Contains(t, buf.String(), "denied")
}

func TestNukeCLIRenderer_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := NewNukeCLIRenderer(&buf)
	assert.NoError(t, r.Render())
	assert.Contains(t, buf.String(), "No resources touched")
}

func TestInspectCLIRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := NewInspectCLIRenderer(&buf)

	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-1", Nukable: true})
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-2", Nukable: false, Reason: "protected"})
	r.OnEvent(reporting.ResourceDeleted{}) // should be ignored

	assert.Len(t, r.resources, 2)

	assert.NoError(t, r.Render())
	assert.Contains(t, buf.String(), "i-1")
	assert.Contains(t, buf.String(), "protected")
}
