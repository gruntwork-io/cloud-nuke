package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedASGroups struct {
	ASGroupsAPI
	DescribeAutoScalingGroupsOutput autoscaling.DescribeAutoScalingGroupsOutput
	DeleteAutoScalingGroupOutput    autoscaling.DeleteAutoScalingGroupOutput
}

func (m mockedASGroups) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return &m.DescribeAutoScalingGroupsOutput, nil
}

func (m mockedASGroups) DeleteAutoScalingGroup(ctx context.Context, params *autoscaling.DeleteAutoScalingGroupInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteAutoScalingGroupOutput, error) {
	return &m.DeleteAutoScalingGroupOutput, nil
}

func TestASGroups_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-asg-1"
	testName2 := "test-asg-2"
	now := time.Now()

	mock := mockedASGroups{
		DescribeAutoScalingGroupsOutput: autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []types.AutoScalingGroup{
				{
					AutoScalingGroupName: aws.String(testName1),
					CreatedTime:          aws.Time(now),
					Tags: []types.TagDescription{
						{Key: aws.String("env"), Value: aws.String("dev")},
					},
				},
				{
					AutoScalingGroupName: aws.String(testName2),
					CreatedTime:          aws.Time(now.Add(1 * time.Hour)),
					Tags: []types.TagDescription{
						{Key: aws.String("env"), Value: aws.String("prod")},
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
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test-asg-1"),
					}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listASGroups(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestASGroups_NukeAll(t *testing.T) {
	t.Parallel()

	mock := mockedASGroups{
		DeleteAutoScalingGroupOutput: autoscaling.DeleteAutoScalingGroupOutput{},
	}

	err := deleteASG(context.Background(), mock, aws.String("test-asg"))
	require.NoError(t, err)
}
