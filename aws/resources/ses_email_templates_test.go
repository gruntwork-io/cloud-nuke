package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSesEmailTemplates_ResourceName(t *testing.T) {
	r := NewSesEmailTemplates()
	assert.Equal(t, "ses-email-template", r.ResourceName())
}

func TestSesEmailTemplates_MaxBatchSize(t *testing.T) {
	r := NewSesEmailTemplates()
	assert.Equal(t, 49, r.MaxBatchSize())
}
