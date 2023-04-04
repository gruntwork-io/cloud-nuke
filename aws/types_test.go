package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewQueryAcceptsValidExcludeAfterEntries(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	type TestCase struct {
		Name                 string
		Regions              []string
		ExcludeRegions       []string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		ExcludeAfter         time.Time
	}

	testCases := []TestCase{
		{
			Name:           "Can pass time.Now plus 24 hours",
			Regions:        []string{"us-east-1"},
			ExcludeRegions: []string{},
			ResourceTypes:  []string{"ec2"},
			ExcludeAfter:   time.Now().Add(time.Hour * 24),
		},
		{
			Name:           "Can pass time.Now",
			Regions:        []string{"us-east-1"},
			ExcludeRegions: []string{},
			ResourceTypes:  []string{"ec2"},
			ExcludeAfter:   time.Now(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewQuery(tc.Regions, tc.ExcludeRegions, tc.ResourceTypes, tc.ExcludeResourceTypes, tc.ExcludeAfter)
			require.NoError(t, err)
		})
	}
}
