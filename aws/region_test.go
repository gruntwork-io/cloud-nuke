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

// A region named twice is one region. The engine refuses a repeated target
// rather than deleting it twice, and the scan and the nuke both read what this
// returns, so narrowing here is what keeps the two from disagreeing.
func TestGetTargetRegionsNamesEachSelectedRegionOnce(t *testing.T) {
	t.Parallel()

	enabled := []string{"us-east-1", "us-west-2", "eu-west-1"}

	regions, err := GetTargetRegions(enabled, []string{"us-east-1", "us-east-1", "eu-west-1"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"us-east-1", "eu-west-1"}, regions)
}
