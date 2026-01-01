package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockCloudtrailClient struct {
	ListTrailsOutput  cloudtrail.ListTrailsOutput
	DeleteTrailOutput cloudtrail.DeleteTrailOutput
	ListTagsOutput    cloudtrail.ListTagsOutput
	ListTagsError     error
}

func (m *mockCloudtrailClient) ListTrails(ctx context.Context, params *cloudtrail.ListTrailsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.ListTrailsOutput, error) {
	return &m.ListTrailsOutput, nil
}

func (m *mockCloudtrailClient) DeleteTrail(ctx context.Context, params *cloudtrail.DeleteTrailInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.DeleteTrailOutput, error) {
	return &m.DeleteTrailOutput, nil
}

func (m *mockCloudtrailClient) ListTags(ctx context.Context, params *cloudtrail.ListTagsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.ListTagsOutput, error) {
	if m.ListTagsError != nil {
		return nil, m.ListTagsError
	}
	// Return tags only for the requested trail ARN
	for _, resourceTag := range m.ListTagsOutput.ResourceTagList {
		if len(params.ResourceIdList) > 0 && *resourceTag.ResourceId == params.ResourceIdList[0] {
			return &cloudtrail.ListTagsOutput{
				ResourceTagList: []types.ResourceTag{resourceTag},
			}, nil
		}
	}
	return &cloudtrail.ListTagsOutput{}, nil
}

func TestListCloudtrailTrails(t *testing.T) {
	t.Parallel()

	testArn1 := "arn:aws:cloudtrail:us-east-1:123456789012:trail/test-trail-1"
	testArn2 := "arn:aws:cloudtrail:us-east-1:123456789012:trail/test-trail-2"
	testName1 := "test-trail-1"
	testName2 := "test-trail-2"

	mock := &mockCloudtrailClient{
		ListTrailsOutput: cloudtrail.ListTrailsOutput{
			Trails: []types.TrailInfo{
				{Name: aws.String(testName1), TrailARN: aws.String(testArn1)},
				{Name: aws.String(testName2), TrailARN: aws.String(testArn2)},
			},
		},
		ListTagsOutput: cloudtrail.ListTagsOutput{
			ResourceTagList: []types.ResourceTag{
				{
					ResourceId: aws.String(testArn1),
					TagsList:   []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
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
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-trail-1")}},
				},
			},
			expected: []string{testArn2},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("prod")}},
				},
			},
			expected: []string{testArn2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			arns, err := listCloudtrailTrails(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestListCloudtrailTrails_TagsError(t *testing.T) {
	t.Parallel()

	testArn := "arn:aws:cloudtrail:us-east-1:123456789012:trail/test-trail"
	mock := &mockCloudtrailClient{
		ListTrailsOutput: cloudtrail.ListTrailsOutput{
			Trails: []types.TrailInfo{{Name: aws.String("test-trail"), TrailARN: aws.String(testArn)}},
		},
		ListTagsError: errors.New("CloudTrailARNInvalidException"),
	}

	arns, err := listCloudtrailTrails(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{testArn}, aws.ToStringSlice(arns))
}

func TestDeleteCloudtrailTrail(t *testing.T) {
	t.Parallel()

	mock := &mockCloudtrailClient{}
	err := deleteCloudtrailTrail(context.Background(), mock, aws.String("arn:aws:cloudtrail:us-east-1:123456789012:trail/test"))
	require.NoError(t, err)
}
