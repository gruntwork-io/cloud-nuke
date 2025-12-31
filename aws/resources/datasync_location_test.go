package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDataSyncLocationClient struct {
	DeleteLocationOutput datasync.DeleteLocationOutput
	ListLocationsOutput  datasync.ListLocationsOutput
}

func (m *mockDataSyncLocationClient) DeleteLocation(ctx context.Context, params *datasync.DeleteLocationInput, optFns ...func(*datasync.Options)) (*datasync.DeleteLocationOutput, error) {
	return &m.DeleteLocationOutput, nil
}

func (m *mockDataSyncLocationClient) ListLocations(ctx context.Context, params *datasync.ListLocationsInput, optFns ...func(*datasync.Options)) (*datasync.ListLocationsOutput, error) {
	return &m.ListLocationsOutput, nil
}

func TestListDataSyncLocations(t *testing.T) {
	t.Parallel()

	testArn1 := fmt.Sprintf("arn::location/test-datasync-location-1")
	testArn2 := fmt.Sprintf("arn::location/test-datasync-location-2")

	mock := &mockDataSyncLocationClient{
		ListLocationsOutput: datasync.ListLocationsOutput{
			Locations: []types.LocationListEntry{
				{LocationArn: aws.String(testArn1)},
				{LocationArn: aws.String(testArn2)},
			},
		},
	}

	arns, err := listDataSyncLocations(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testArn1, testArn2}, aws.ToStringSlice(arns))
}

func TestDeleteDataSyncLocation(t *testing.T) {
	t.Parallel()

	mock := &mockDataSyncLocationClient{}
	err := deleteDataSyncLocation(context.Background(), mock, aws.String("arn::location/test"))
	require.NoError(t, err)
}
