package renderers

import (
	"bytes"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/stretchr/testify/assert"
)

func TestCLIRenderer_Nuke(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// Send nuke events
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Identifier: "i-1", Success: true})
	r.OnEvent(reporting.ResourceDeleted{ResourceType: "ec2", Identifier: "i-2", Success: false, Error: "denied"})
	r.OnEvent(reporting.GeneralError{ResourceType: "s3", Description: "list failed", Error: "timeout"})

	assert.Len(t, r.deleted, 2)
	assert.Len(t, r.errors, 1)

	assert.NoError(t, r.Render())
	assert.Contains(t, buf.String(), "i-1")
	assert.Contains(t, buf.String(), "denied")
}

func TestCLIRenderer_Inspect(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)

	// Send inspect events
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-1", Nukable: true})
	r.OnEvent(reporting.ResourceFound{ResourceType: "ec2", Region: "us-east-1", Identifier: "i-2", Nukable: false, Reason: "protected"})

	assert.Len(t, r.found, 2)

	assert.NoError(t, r.Render())
	assert.Contains(t, buf.String(), "i-1")
	assert.Contains(t, buf.String(), "protected")
}

func TestCLIRenderer_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := NewCLIRenderer(&buf)
	assert.NoError(t, r.Render())
	assert.Contains(t, buf.String(), "No resources found")
}
