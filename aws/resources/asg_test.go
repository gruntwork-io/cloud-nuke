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
	"github.com/stretchr/testify/assert"
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

func TestAutoScalingGroupGetAll(t *testing.T) {
	t.Parallel()

	testName := "cloud-nuke-test"
	now := time.Now()
	ag := ASGroups{
		Client: mockedASGroups{
			DescribeAutoScalingGroupsOutput: autoscaling.DescribeAutoScalingGroupsOutput{
				AutoScalingGroups: []types.AutoScalingGroup{{
					AutoScalingGroupName: aws.String(testName),
					CreatedTime:          aws.Time(now),
				}}}}}

	// empty filter
	groups, err := ag.getAll(context.Background(), config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(groups), testName)

	// name filter
	groups, err = ag.getAll(context.Background(), config.Config{
		AutoScalingGroup: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("^cloud-nuke-*"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(groups), testName)

	// time filter
	groups, err = ag.getAll(context.Background(), config.Config{
		AutoScalingGroup: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-1)),
			}}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(groups), testName)
}

func TestAutoScalingGroupNukeAll(t *testing.T) {
	t.Parallel()

	ag := ASGroups{
		BaseAwsResource: BaseAwsResource{
			Context: context.Background(),
			Timeout: DefaultWaitTimeout,
		},
		Client: mockedASGroups{
			DeleteAutoScalingGroupOutput: autoscaling.DeleteAutoScalingGroupOutput{},
		},
	}

	err := ag.nukeAll([]*string{aws.String("cloud-nuke-test")})
	assert.NoError(t, err)
}
