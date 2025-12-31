package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEC2PlacementGroupsClient struct {
	DescribePlacementGroupsOutput ec2.DescribePlacementGroupsOutput
	DeletePlacementGroupOutput    ec2.DeletePlacementGroupOutput
	CreateTagsOutput              ec2.CreateTagsOutput
}

func (m *mockEC2PlacementGroupsClient) DescribePlacementGroups(ctx context.Context, params *ec2.DescribePlacementGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribePlacementGroupsOutput, error) {
	return &m.DescribePlacementGroupsOutput, nil
}

func (m *mockEC2PlacementGroupsClient) DeletePlacementGroup(ctx context.Context, params *ec2.DeletePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeletePlacementGroupOutput, error) {
	return &m.DeletePlacementGroupOutput, nil
}

func (m *mockEC2PlacementGroupsClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &m.CreateTagsOutput, nil
}

func TestEC2PlacementGroups_ResourceName(t *testing.T) {
	r := NewEC2PlacementGroups()
	assert.Equal(t, "ec2-placement-groups", r.ResourceName())
}

func TestEC2PlacementGroups_MaxBatchSize(t *testing.T) {
	r := NewEC2PlacementGroups()
	assert.Equal(t, 200, r.MaxBatchSize())
}

func TestListEC2PlacementGroups(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	mock := &mockEC2PlacementGroupsClient{
		DescribePlacementGroupsOutput: ec2.DescribePlacementGroupsOutput{
			PlacementGroups: []types.PlacementGroup{
				{
					GroupName: aws.String("pg1"),
					GroupId:   aws.String("pg-123"),
					Tags:      []types.Tag{{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))}},
				},
				{
					GroupName: aws.String("pg2"),
					GroupId:   aws.String("pg-456"),
					Tags:      []types.Tag{{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))}},
				},
			},
		},
	}

	names, err := listEC2PlacementGroups(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"pg1", "pg2"}, aws.ToStringSlice(names))
}

func TestListEC2PlacementGroups_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	mock := &mockEC2PlacementGroupsClient{
		DescribePlacementGroupsOutput: ec2.DescribePlacementGroupsOutput{
			PlacementGroups: []types.PlacementGroup{
				{
					GroupName: aws.String("pg1"),
					GroupId:   aws.String("pg-123"),
					Tags:      []types.Tag{{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))}},
				},
				{
					GroupName: aws.String("skip-this"),
					GroupId:   aws.String("pg-456"),
					Tags:      []types.Tag{{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))}},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listEC2PlacementGroups(ctx, mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"pg1"}, aws.ToStringSlice(names))
}

func TestDeleteEC2PlacementGroup(t *testing.T) {
	t.Parallel()

	mock := &mockEC2PlacementGroupsClient{}
	err := deleteEC2PlacementGroup(context.Background(), mock, aws.String("test-pg"))
	require.NoError(t, err)
}
