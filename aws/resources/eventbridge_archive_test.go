package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedEventBridgeArchiveService struct {
	EventBridgeArchiveAPI
	DeleteArchiveOutput   eventbridge.DeleteArchiveOutput
	ListArchivesOutput    eventbridge.ListArchivesOutput
	DescribeArchiveByName map[string]eventbridge.DescribeArchiveOutput
	TagsByArn             map[string][]types.Tag
}

func (m mockedEventBridgeArchiveService) DeleteArchive(ctx context.Context, params *eventbridge.DeleteArchiveInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteArchiveOutput, error) {
	return &m.DeleteArchiveOutput, nil
}

func (m mockedEventBridgeArchiveService) ListArchives(ctx context.Context, params *eventbridge.ListArchivesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListArchivesOutput, error) {
	return &m.ListArchivesOutput, nil
}

func (m mockedEventBridgeArchiveService) DescribeArchive(ctx context.Context, params *eventbridge.DescribeArchiveInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeArchiveOutput, error) {
	if m.DescribeArchiveByName != nil {
		if out, ok := m.DescribeArchiveByName[aws.ToString(params.ArchiveName)]; ok {
			return &out, nil
		}
	}
	return &eventbridge.DescribeArchiveOutput{}, nil
}

func (m mockedEventBridgeArchiveService) ListTagsForResource(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error) {
	if m.TagsByArn != nil {
		if tags, ok := m.TagsByArn[aws.ToString(params.ResourceARN)]; ok {
			return &eventbridge.ListTagsForResourceOutput{Tags: tags}, nil
		}
	}
	return &eventbridge.ListTagsForResourceOutput{}, nil
}

func Test_EventBridgeArchive_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	archive1 := "test-archive-1"
	archive2 := "test-archive-2"
	archive1Arn := "arn:aws:events:us-east-1:123456789012:archive/test-archive-1"
	archive2Arn := "arn:aws:events:us-east-1:123456789012:archive/test-archive-2"

	client := mockedEventBridgeArchiveService{
		ListArchivesOutput: eventbridge.ListArchivesOutput{
			Archives: []types.Archive{
				{
					ArchiveName:  aws.String(archive1),
					CreationTime: &now,
				},
				{
					ArchiveName:  aws.String(archive2),
					CreationTime: aws.Time(now.Add(time.Hour)),
				},
			},
		},
		DescribeArchiveByName: map[string]eventbridge.DescribeArchiveOutput{
			archive1: {ArchiveArn: aws.String(archive1Arn)},
			archive2: {ArchiveArn: aws.String(archive2Arn)},
		},
		TagsByArn: map[string][]types.Tag{
			archive1Arn: {{Key: aws.String("env"), Value: aws.String("production")}},
			archive2Arn: {{Key: aws.String("env"), Value: aws.String("development")}},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{archive1, archive2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(archive1),
					}},
				}},
			expected: []string{archive2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^development$")},
					},
				}},
			expected: []string{archive2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			archives, err := listEventBridgeArchives(
				context.Background(),
				client,
				resource.Scope{},
				tc.configObj,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(archives))
		})
	}
}

func Test_EventBridgeArchive_NukeAll(t *testing.T) {
	t.Parallel()

	archiveName := "test-archive-1"
	client := mockedEventBridgeArchiveService{
		DeleteArchiveOutput: eventbridge.DeleteArchiveOutput{},
	}

	err := deleteEventBridgeArchive(context.Background(), client, &archiveName)
	assert.NoError(t, err)
}
