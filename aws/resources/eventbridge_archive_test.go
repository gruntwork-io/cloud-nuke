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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedEventBridgeArchiveService struct {
	EventBridgeArchiveAPI
	DeleteArchiveOutput eventbridge.DeleteArchiveOutput
	ListArchivesOutput  eventbridge.ListArchivesOutput
}

func (m mockedEventBridgeArchiveService) DeleteArchive(ctx context.Context, params *eventbridge.DeleteArchiveInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteArchiveOutput, error) {
	return &m.DeleteArchiveOutput, nil
}

func (m mockedEventBridgeArchiveService) ListArchives(ctx context.Context, params *eventbridge.ListArchivesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListArchivesOutput, error) {
	return &m.ListArchivesOutput, nil
}

func Test_EventBridgeArchive_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()

	archive1 := "test-archive-1"
	archive2 := "test-archive-2"

	service := EventBridgeArchive{
		Client: mockedEventBridgeArchiveService{
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			buses, err := service.getAll(
				context.Background(),
				config.Config{EventBridgeArchive: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(buses))
		})
	}
}

func Test_EventBridgeArchive_NukeAll(t *testing.T) {
	t.Parallel()

	archiveName := "test-archive-1"
	service := EventBridgeArchive{
		Client: mockedEventBridgeArchiveService{DeleteArchiveOutput: eventbridge.DeleteArchiveOutput{}},
	}

	err := service.nukeAll([]*string{&archiveName})
	assert.NoError(t, err)
}
