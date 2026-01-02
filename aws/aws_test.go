package aws

import (
	"os"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
)

// hasAWSCredentials checks if AWS credentials are available via environment variables.
// This is a fast check that doesn't require calling AWS APIs.
func hasAWSCredentials() bool {
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

func TestSplit(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testCases := []struct {
		limit    int
		array    []string
		expected [][]string
	}{
		{2, []string{"a", "b", "c", "d"}, [][]string{{"a", "b"}, {"c", "d"}}},
		{3, []string{"a", "b", "c", "d"}, [][]string{{"a", "b", "c"}, {"d"}}},
		{2, []string{"a", "b", "c"}, [][]string{{"a", "b"}, {"c"}}},
		{5, []string{"a", "b", "c"}, [][]string{{"a", "b", "c"}}},
		{-2, []string{"a", "b", "c"}, [][]string{{"a", "b"}, {"c"}}},
		{0, []string{"a", "b", "c"}, [][]string{{"a", "b", "c"}}},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.expected, util.Split(testCase.array, testCase.limit))
	}
}

func TestGetTargetRegions(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	// Skip if AWS credentials are not available (fast check via env vars)
	if !hasAWSCredentials() {
		t.Skip("Skipping test: AWS credentials not available")
	}

	actualEnabledRegions, err := GetEnabledRegions()
	if err != nil {
		t.Skipf("Skipping test: failed to get enabled regions: %v", err)
	}
	assert.Greater(t, len(actualEnabledRegions), 0)

	type test struct {
		enabledRegions  []string
		selectedRegions []string
		excludedRegions []string
		outputRegions   []string
	}

	testEnabledRegions := []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"}

	validInputTests := []test{
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{},
			excludedRegions: []string{},
			outputRegions:   testEnabledRegions,
		},
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{"us-east-1"},
			excludedRegions: []string{},
			outputRegions:   []string{"us-east-1"},
		},
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{"us-east-1", "us-east-2"},
			excludedRegions: []string{},
			outputRegions:   []string{"us-east-1", "us-east-2"},
		},
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{},
			excludedRegions: []string{"us-east-1"},
			outputRegions:   []string{"us-east-2", "us-west-1", "us-west-2"},
		},
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{},
			excludedRegions: []string{"us-east-1", "us-east-2"},
			outputRegions:   []string{"us-west-1", "us-west-2"},
		},
	}

	for _, testCase := range validInputTests {
		outputRegions, err := GetTargetRegions(testCase.enabledRegions, testCase.selectedRegions, testCase.excludedRegions)
		assert.Equal(t, outputRegions, testCase.outputRegions)
		assert.Equal(t, err, nil)
	}

	invalidInputTests := []test{
		// Cannot specify empty enabledRegions
		test{
			enabledRegions:  []string{},
			selectedRegions: []string{"us-east-1"},
			excludedRegions: []string{"us-west-1"},
			outputRegions:   nil,
		},
		// Cannot specify both selectedRegions and excludedRegions
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{"us-east-1"},
			excludedRegions: []string{"us-west-1"},
			outputRegions:   nil,
		},
		// Cannot specify invalid selectedRegion
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{"us-east-1", "xyz"},
			excludedRegions: []string{},
			outputRegions:   nil,
		},
		// Cannot specify invalid excludedRegion
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{},
			excludedRegions: []string{"us-east-1", "xyz"},
			outputRegions:   nil,
		},
		// Cannot exclude all regions
		test{
			enabledRegions:  testEnabledRegions,
			selectedRegions: []string{},
			excludedRegions: testEnabledRegions,
			outputRegions:   nil,
		},
	}

	for _, testCase := range invalidInputTests {
		outputRegions, err := GetTargetRegions(testCase.enabledRegions, testCase.selectedRegions, testCase.excludedRegions)
		assert.Equal(t, outputRegions, testCase.outputRegions)
		assert.NotEqual(t, err, nil)
	}
}
