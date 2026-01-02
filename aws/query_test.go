package aws

import (
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gruntwork-io/cloud-nuke/telemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hasAWSCredentialsForQuery checks if AWS credentials are available via environment variables.
// This is a fast check that doesn't require calling AWS APIs.
func hasAWSCredentialsForQuery() bool {
	// Check for standard AWS credential environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}
	// Check for AWS profile
	if os.Getenv("AWS_PROFILE") != "" {
		return true
	}
	// Check for web identity token (used in EKS/CI)
	if os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" {
		return true
	}
	return false
}

func TestNewQueryAcceptsValidExcludeAfterEntries(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")

	// Skip if AWS credentials are not available (fast check via env vars)
	// NewQuery validates regions against AWS, which requires credentials
	if !hasAWSCredentialsForQuery() {
		t.Skip("Skipping test: AWS credentials not available")
	}

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
			q, err := NewQuery(
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
			assert.True(t, q.ProtectUntilExpire)
		})
	}
}
