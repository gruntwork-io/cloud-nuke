package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDatasyncLocation struct {
	DataSyncLocationAPI
	datasync.DeleteLocationOutput
	datasync.ListLocationsOutput
}

func (m mockDatasyncLocation) DeleteLocation(ctx context.Context, params *datasync.DeleteLocationInput, optFns ...func(*datasync.Options)) (*datasync.DeleteLocationOutput, error) {
	return &m.DeleteLocationOutput, nil
}

func (m mockDatasyncLocation) ListLocations(ctx context.Context, params *datasync.ListLocationsInput, optFns ...func(*datasync.Options)) (*datasync.ListLocationsOutput, error) {
	return &m.ListLocationsOutput, nil
}

func TestDataSyncLocation_NukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-datasync-location"
	service := DataSyncLocation{
		Client: mockDatasyncLocation{
			DeleteLocationOutput: datasync.DeleteLocationOutput{},
		},
	}

	err := service.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}

func TestDataSyncLocation_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-datasync-location-1"
	testName2 := "test-datasync-location-2"
	location := DataSyncLocation{
		Client: mockDatasyncLocation{
			ListLocationsOutput: datasync.ListLocationsOutput{
				Locations: []types.LocationListEntry{
					{
						LocationArn: aws.String(fmt.Sprintf("arn::location/%s", testName1)),
					},
					{
						LocationArn: aws.String(fmt.Sprintf("arn::location/%s", testName2)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{fmt.Sprintf("arn::location/%s", testName1), fmt.Sprintf("arn::location/%s", testName2)},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := location.getAll(context.Background(), config.Config{
				DataSyncLocation: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}
