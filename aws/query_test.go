package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gruntwork-io/cloud-nuke/telemetry"

	"github.com/stretchr/testify/require"
)

func TestNewQueryAcceptsValidExcludeAfterEntries(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	type TestCase struct {
		Name                 string
		Regions              []string
		ExcludeRegions       []string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		ExcludeAfter         *time.Time
		IncludeAfter         *time.Time
	}

	testCases := []TestCase{
		{
			Name:           "Can pass time.Now plus 24 hours",
			Regions:        []string{"us-east-1"},
			ExcludeRegions: []string{},
			ResourceTypes:  []string{"ec2"},
			ExcludeAfter:   aws.Time(time.Now().Add(time.Hour * 24)),
			IncludeAfter:   aws.Time(time.Now().Add(time.Hour * 24)),
		},
		{
			Name:           "Can pass time.Now",
			Regions:        []string{"us-east-1"},
			ExcludeRegions: []string{},
			ResourceTypes:  []string{"ec2"},
			ExcludeAfter:   aws.Time(time.Now()),
			IncludeAfter:   aws.Time(time.Now()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewQuery(
				tc.Regions,
				tc.ExcludeRegions,
				tc.ResourceTypes,
				tc.ExcludeResourceTypes,
				tc.ExcludeAfter,
				tc.IncludeAfter,
				false,
				nil,
				false,
				false,
			)
			require.NoError(t, err)
		})
	}
}
