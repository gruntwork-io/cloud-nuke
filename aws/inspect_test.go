package aws

import (
	"reflect"
	"testing"

	"github.com/andrewderr/cloud-nuke-a1/telemetry"

	"github.com/stretchr/testify/require"
)

func TestHandleResourceTypeSelectionsRejectsInvalid(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	type TestCase struct {
		Name                 string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		Want                 []string
		Error                InvalidResourceTypesSuppliedError
	}

	testCases := []TestCase{
		{
			Name:                 "Invalid resource type is rejected",
			ResourceTypes:        []string{"invalid_resource"},
			ExcludeResourceTypes: []string{},
			Want:                 []string{},
			Error:                InvalidResourceTypesSuppliedError{},
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
		Error                ResourceTypeAndExcludeFlagsBothPassedError
	}

	testCases := []TestCase{
		{
			Name:                 "Valid resources and valid excludes result in filtering",
			ResourceTypes:        []string{"ec2", "s3", "lambda"},
			ExcludeResourceTypes: []string{"ec2"},
			Want:                 []string{},
			Error:                ResourceTypeAndExcludeFlagsBothPassedError{},
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
