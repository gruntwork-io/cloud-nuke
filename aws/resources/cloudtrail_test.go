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

	testName1 := "test-name1"
	testName2 := "test-name2"
	testArn1 := "test-arn1"
	testArn2 := "test-arn2"

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
					TagsList: []types.Tag{
						{Key: aws.String("t_name"), Value: &testName1},
						{Key: aws.String("t_arn"), Value: &testArn1},
					},
				},
			},
		},
	}

	arns, err := listCloudtrailTrails(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testArn1, testArn2}, aws.ToStringSlice(arns))
}

func TestListCloudtrailTrails_WithFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "skip-this"
	testArn1 := "test-arn1"
	testArn2 := "test-arn2"

	mock := &mockCloudtrailClient{
		ListTrailsOutput: cloudtrail.ListTrailsOutput{
			Trails: []types.TrailInfo{
				{Name: aws.String(testName1), TrailARN: aws.String(testArn1)},
				{Name: aws.String(testName2), TrailARN: aws.String(testArn2)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	arns, err := listCloudtrailTrails(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testArn1}, aws.ToStringSlice(arns))
}

func TestListCloudtrailTrails_WithTagsError(t *testing.T) {
	t.Parallel()

	mock := &mockCloudtrailClient{
		ListTrailsOutput: cloudtrail.ListTrailsOutput{
			Trails: []types.TrailInfo{{
				Name:     aws.String("test-trail"),
				TrailARN: aws.String("test-arn"),
			}},
		},
		ListTagsError: errors.New("CloudTrailARNInvalidException: You cannot have resources belonging to multiple owners"),
	}

	arns, err := listCloudtrailTrails(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"test-arn"}, aws.ToStringSlice(arns))
}

func TestDeleteCloudtrailTrail(t *testing.T) {
	t.Parallel()

	mock := &mockCloudtrailClient{}
	err := deleteCloudtrailTrail(context.Background(), mock, aws.String("test-arn"))
	require.NoError(t, err)
}
