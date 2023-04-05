package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
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
		assert.Equal(t, testCase.expected, split(testCase.array, testCase.limit))
	}
}

func TestGetTargetRegions(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	actualEnabledRegions, _ := GetEnabledRegions()
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
