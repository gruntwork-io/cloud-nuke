package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalRegion(t *testing.T) {
	t.Run("NoEnv", func(t *testing.T) {
		c, err := NewSession(GlobalRegion)
		require.NoError(t, err)
		assert.Equal(t, DefaultRegion, c.Region)

	})

	t.Run("WithEnv", func(t *testing.T) {
		global := "us-gov-east-1"
		t.Setenv("CLOUD_NUKE_AWS_GLOBAL_REGION", global)
		c, err := NewSession(GlobalRegion)
		require.NoError(t, err)
		assert.Equal(t, global, c.Region)
	})

}
