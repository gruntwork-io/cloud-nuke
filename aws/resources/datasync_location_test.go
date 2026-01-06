package resources

import (
	"context"
	"regexp"
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

	testArn1 := "arn:aws:datasync:us-east-1:123456789012:location/loc-1234567890abcdef0"
	testArn2 := "arn:aws:datasync:us-east-1:123456789012:location/loc-0987654321fedcba0"
	testUri1 := "s3://my-bucket/prefix"
	testUri2 := "nfs://192.168.1.100/exports/data"

	mock := &mockDataSyncLocationClient{
		ListLocationsOutput: datasync.ListLocationsOutput{
			Locations: []types.LocationListEntry{
				{LocationArn: aws.String(testArn1), LocationUri: aws.String(testUri1)},
				{LocationArn: aws.String(testArn2), LocationUri: aws.String(testUri2)},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("s3://")}},
				},
			},
			expected: []string{testArn2},
		},
		"nameInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("nfs://")}},
				},
			},
			expected: []string{testArn2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			arns, err := listDataSyncLocations(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestDeleteDataSyncLocation(t *testing.T) {
	t.Parallel()

	mock := &mockDataSyncLocationClient{}
	err := deleteDataSyncLocation(context.Background(), mock, aws.String("arn:aws:datasync:us-east-1:123456789012:location/loc-test"))
	require.NoError(t, err)
}
