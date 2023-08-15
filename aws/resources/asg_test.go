package resources

import (
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

type mockedASGroups struct {
	autoscalingiface.AutoScalingAPI
	DescribeAutoScalingGroupsResp autoscaling.DescribeAutoScalingGroupsOutput
	DeleteAutoScalingGroupResp    autoscaling.DeleteAutoScalingGroupOutput
}

func (m mockedASGroups) DescribeAutoScalingGroups(input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return &m.DescribeAutoScalingGroupsResp, nil
}

func (m mockedASGroups) DeleteAutoScalingGroup(input *autoscaling.DeleteAutoScalingGroupInput) (*autoscaling.DeleteAutoScalingGroupOutput, error) {
	return &m.DeleteAutoScalingGroupResp, nil
}

func (m mockedASGroups) WaitUntilGroupNotExists(input *autoscaling.DescribeAutoScalingGroupsInput) error {
	return nil
}

func TestAutoScalingGroupGetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName := "cloud-nuke-test"
	now := time.Now()
	ag := ASGroups{
		Client: mockedASGroups{
			DescribeAutoScalingGroupsResp: autoscaling.DescribeAutoScalingGroupsOutput{
				AutoScalingGroups: []*autoscaling.Group{{
					AutoScalingGroupName: awsgo.String(testName),
					CreatedTime:          awsgo.Time(now),
				}}}}}

	// empty filter
	groups, err := ag.getAll(config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(groups), testName)

	// name filter
	groups, err = ag.getAll(config.Config{
		AutoScalingGroup: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("^cloud-nuke-*"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(groups), testName)

	// time filter
	groups, err = ag.getAll(config.Config{
		AutoScalingGroup: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: awsgo.Time(now.Add(-1)),
			}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(groups), testName)
}

func TestAutoScalingGroupNukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ag := ASGroups{
		Client: mockedASGroups{
			DeleteAutoScalingGroupResp: autoscaling.DeleteAutoScalingGroupOutput{},
		}}

	err := ag.nukeAll([]*string{awsgo.String("cloud-nuke-test")})
	assert.NoError(t, err)
}
