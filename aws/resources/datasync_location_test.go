package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDataSyncLocationClient struct {
	DataSyncLocationAPI
	DeleteLocationOutput datasync.DeleteLocationOutput
	ListLocationsOutput  datasync.ListLocationsOutput
	CreationTimes        map[string]time.Time
	DescribeCalls        int
}

func (m *mockDataSyncLocationClient) creationTime(arn *string) (*time.Time, error) {
	m.DescribeCalls++
	createdAt, ok := m.CreationTimes[aws.ToString(arn)]
	if !ok {
		return nil, errors.New("access denied")
	}
	return aws.Time(createdAt), nil
}

func (m *mockDataSyncLocationClient) DescribeLocationS3(ctx context.Context, params *datasync.DescribeLocationS3Input, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationS3Output, error) {
	createdAt, err := m.creationTime(params.LocationArn)
	if err != nil {
		return nil, err
	}
	return &datasync.DescribeLocationS3Output{CreationTime: createdAt}, nil
}

func (m *mockDataSyncLocationClient) DescribeLocationNfs(ctx context.Context, params *datasync.DescribeLocationNfsInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationNfsOutput, error) {
	createdAt, err := m.creationTime(params.LocationArn)
	if err != nil {
		return nil, err
	}
	return &datasync.DescribeLocationNfsOutput{CreationTime: createdAt}, nil
}

func (m *mockDataSyncLocationClient) DeleteLocation(ctx context.Context, params *datasync.DeleteLocationInput, optFns ...func(*datasync.Options)) (*datasync.DeleteLocationOutput, error) {
	return &m.DeleteLocationOutput, nil
}

func (m *mockDataSyncLocationClient) ListLocations(ctx context.Context, params *datasync.ListLocationsInput, optFns ...func(*datasync.Options)) (*datasync.ListLocationsOutput, error) {
	return &m.ListLocationsOutput, nil
}

func (m *mockDataSyncLocationClient) ListTagsForResource(ctx context.Context, params *datasync.ListTagsForResourceInput, optFns ...func(*datasync.Options)) (*datasync.ListTagsForResourceOutput, error) {
	if aws.ToString(params.ResourceArn) == "arn:aws:datasync:us-east-1:123456789012:location/loc-1234567890abcdef0" {
		return &datasync.ListTagsForResourceOutput{
			Tags: []types.TagListEntry{{Key: aws.String("env"), Value: aws.String("prod")}},
		}, nil
	}
	return &datasync.ListTagsForResourceOutput{}, nil
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
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^prod$")}},
				},
			},
			expected: []string{testArn1},
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

func TestListDataSyncLocations_TimeFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	oldS3 := "arn:aws:datasync:us-east-1:123456789012:location/loc-old"
	newNfs := "arn:aws:datasync:us-east-1:123456789012:location/loc-new"
	unreadableNfs := "arn:aws:datasync:us-east-1:123456789012:location/loc-unreadable"
	unknownType := "arn:aws:datasync:us-east-1:123456789012:location/loc-unknown"
	newMock := func() *mockDataSyncLocationClient {
		return &mockDataSyncLocationClient{
			ListLocationsOutput: datasync.ListLocationsOutput{
				Locations: []types.LocationListEntry{
					{LocationArn: aws.String(oldS3), LocationUri: aws.String("s3://my-bucket/prefix/")},
					{LocationArn: aws.String(newNfs), LocationUri: aws.String("nfs://192.168.1.100/exports/")},
					{LocationArn: aws.String(unreadableNfs), LocationUri: aws.String("nfs://192.168.1.101/exports/")},
					{LocationArn: aws.String(unknownType), LocationUri: aws.String("future-type://host/path/")},
				},
			},
			CreationTimes: map[string]time.Time{
				oldS3:  now.Add(-2 * time.Hour),
				newNfs: now,
			},
		}
	}

	mock := newMock()
	olderThanAnHour := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-time.Hour))}}
	arns, err := listDataSyncLocations(context.Background(), mock, resource.Scope{}, olderThanAnHour)
	require.NoError(t, err)
	require.Equal(t, []string{oldS3}, aws.ToStringSlice(arns), "an unreadable or unknown location type leaves the location out")
	require.Equal(t, 3, mock.DescribeCalls, "the URI scheme picks the Describe call, and an unknown scheme makes none")

	mock = newMock()
	olderThanAnHourExceptNew := olderThanAnHour
	olderThanAnHourExceptNew.ExcludeRule.NamesRegExp = []config.Expression{{RE: *regexp.MustCompile(`192\.168\.1\.100`)}}
	_, err = listDataSyncLocations(context.Background(), mock, resource.Scope{}, olderThanAnHourExceptNew)
	require.NoError(t, err)
	require.Equal(t, 2, mock.DescribeCalls, "a location excluded by name gets no Describe call")

	mock = newMock()
	arns, err = listDataSyncLocations(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Len(t, arns, 4)
	require.Zero(t, mock.DescribeCalls, "no time filter, no Describe calls")
}

func TestDeleteDataSyncLocation(t *testing.T) {
	t.Parallel()

	mock := &mockDataSyncLocationClient{}
	err := deleteDataSyncLocation(context.Background(), mock, aws.String("arn:aws:datasync:us-east-1:123456789012:location/loc-test"))
	require.NoError(t, err)
}
