package aws

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewQueryRejectsInvalidExcludeAfterEntries(t *testing.T) {

	type TestCase struct {
		Name                 string
		Regions              []string
		ExcludeRegions       []string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		ExcludeAfter         string
		Error                InvalidTimeStringPassedError
	}

	testCases := []TestCase{
		{
			Name:                 "Invalid excludeAfter time string rejected",
			Regions:              []string{"us-east-1"},
			ExcludeRegions:       []string{},
			ResourceTypes:        []string{"ec2", "vpc"},
			ExcludeResourceTypes: []string{},
			ExcludeAfter:         "this is not a valid time duration",
			Error:                InvalidTimeStringPassedError{},
		},
		{
			Name:                 "Empty excludeAfter time string rejected",
			Regions:              []string{"us-east-1"},
			ExcludeRegions:       []string{},
			ResourceTypes:        []string{"ec2", "vpc"},
			ExcludeResourceTypes: []string{},
			ExcludeAfter:         "", // this ExcludeAfter is intentionally left empty
			Error:                InvalidTimeStringPassedError{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewQuery(tc.Regions, tc.ExcludeRegions, tc.ResourceTypes, tc.ExcludeResourceTypes, tc.ExcludeAfter)
			if !errors.As(err, &tc.Error) {
				t.Logf("%s: Expected error of type %T but got %T", tc.Name, tc.Error, err)
				t.Fail()
			}
		})
	}
}

func TestNewQueryAcceptsValidExcludeAfterEntries(t *testing.T) {
	type TestCase struct {
		Name                 string
		Regions              []string
		ExcludeRegions       []string
		ResourceTypes        []string
		ExcludeResourceTypes []string
		ExcludeAfter         string
	}

	testCases := []TestCase{
		{
			Name:           "Can pass simple durations: 1h",
			Regions:        []string{"us-east-1"},
			ExcludeRegions: []string{},
			ResourceTypes:  []string{"ec2"},
			ExcludeAfter:   "1h",
		},
		{
			Name:           "Can pass simple durations: 65m",
			Regions:        []string{"us-east-1"},
			ExcludeRegions: []string{},
			ResourceTypes:  []string{"ec2"},
			ExcludeAfter:   "65m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewQuery(tc.Regions, tc.ExcludeRegions, tc.ResourceTypes, tc.ExcludeResourceTypes, tc.ExcludeAfter)
			require.NoError(t, err)
		})
	}
}
