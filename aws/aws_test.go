package aws

import (
	"reflect"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestHandleResourceTypeSelectionsRejectsInvalid(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	type TestCase struct {
		Name                 string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		Want                 []string
		Error                resource.InvalidResourceTypesSuppliedError
	}

	testCases := []TestCase{
		{
			Name:                 "Invalid resource type is rejected",
			ResourceTypes:        []string{"invalid_resource"},
			ExcludeResourceTypes: []string{},
			Want:                 []string{},
			Error:                resource.InvalidResourceTypesSuppliedError{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := HandleResourceTypeSelections(tc.ResourceTypes, tc.ExcludeResourceTypes)
			require.Error(t, err)
			require.ErrorAs(t, err, &tc.Error)
		})
	}

}

func TestHandleResourceTypeSelectionsRejectsConflictingParams(t *testing.T) {
	type TestCase struct {
		Name                 string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		Want                 []string
		Error                resource.ResourceTypeAndExcludeFlagsBothPassedError
	}

	testCases := []TestCase{
		{
			Name:                 "Valid resources and valid excludes result in filtering",
			ResourceTypes:        []string{"ec2", "s3", "lambda"},
			ExcludeResourceTypes: []string{"ec2"},
			Want:                 []string{},
			Error:                resource.ResourceTypeAndExcludeFlagsBothPassedError{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := HandleResourceTypeSelections(tc.ResourceTypes, tc.ExcludeResourceTypes)
			require.Error(t, err)
			require.ErrorAs(t, err, &tc.Error)
		})
	}

}

func TestHandleResourceTypeSelectionsFiltering(t *testing.T) {
	type TestCase struct {
		Name                 string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		Want                 []string
	}

	testCases := []TestCase{{
		Name:                 "Valid resource types are accepted",
		ResourceTypes:        []string{"ec2", "vpc"},
		ExcludeResourceTypes: []string{},
		Want:                 []string{"ec2", "vpc"},
	},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := HandleResourceTypeSelections(tc.ResourceTypes, tc.ExcludeResourceTypes)
			require.NoError(t, err)
			if !reflect.DeepEqual(got, tc.Want) {
				t.Logf("%s: Expected %v but got %v", tc.Name, tc.Want, got)
				t.Fail()
			}
		})
	}
}
