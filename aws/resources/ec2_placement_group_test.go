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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEC2PlacementGroups struct {
	EC2PlacementGroupsAPI
	DescribePlacementGroupsOutput ec2.DescribePlacementGroupsOutput
	DeletePlacementGroupOutput    ec2.DeletePlacementGroupOutput
}

func (m mockedEC2PlacementGroups) DescribePlacementGroups(ctx context.Context, params *ec2.DescribePlacementGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribePlacementGroupsOutput, error) {
	return &m.DescribePlacementGroupsOutput, nil
}

func (m mockedEC2PlacementGroups) DeletePlacementGroup(ctx context.Context, params *ec2.DeletePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeletePlacementGroupOutput, error) {
	return &m.DeletePlacementGroupOutput, nil
}

func TestEC2PlacementGroups_GetAll(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	now := time.Now()
	testId1 := "test-group-id1"
	testName1 := "test-group1"
	testId2 := "test-group-id2"
	testName2 := "test-group2"
	p := EC2PlacementGroups{
		Client: mockedEC2PlacementGroups{
			DescribePlacementGroupsOutput: ec2.DescribePlacementGroupsOutput{
				PlacementGroups: []types.PlacementGroup{
					{
						GroupName: aws.String(testName1),
						GroupId:   aws.String(testId1),
						Tags: []types.Tag{{
							Key:   aws.String(util.FirstSeenTagKey),
							Value: aws.String(util.FormatTimestamp(now)),
						}},
					},
					{
						GroupName: aws.String(testName2),
						GroupId:   aws.String(testId2),
						Tags: []types.Tag{{
							Key:   aws.String(util.FirstSeenTagKey),
							Value: aws.String(util.FormatTimestamp(now.Add(2 * time.Hour))),
						}},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(1 * time.Hour)),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := p.getAll(tc.ctx, config.Config{
				EC2PlacementGroups: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEC2PlacementGroups_NukeAll(t *testing.T) {
	t.Parallel()
	h := EC2PlacementGroups{
		Client: mockedEC2PlacementGroups{
			DeletePlacementGroupOutput: ec2.DeletePlacementGroupOutput{},
		},
	}

	err := h.nukeAll([]*string{aws.String("test-group1"), aws.String("test-group2")})
	require.NoError(t, err)
}
