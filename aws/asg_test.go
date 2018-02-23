package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/aws-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestAutoScalingGroup(t *testing.T, session *session.Session, name string) {
	svc := autoscaling.New(session)
	instance := createTestEC2Instance(t, session, name, false)

	param := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: &name,
		InstanceId:           instance.InstanceId,
		MinSize:              awsgo.Int64(1),
		MaxSize:              awsgo.Int64(2),
	}

	_, err := svc.CreateAutoScalingGroup(param)
	if err != nil {
		assert.Failf(t, "Could not create test ASG: %s", errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilGroupExists(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&name},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
}

func TestListAutoScalingGroups(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	createTestAutoScalingGroup(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllAutoScalingGroups(session, []*string{&uniqueTestID})
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

	groupNames, err := getAllAutoScalingGroups(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Auto Scaling Groups")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)

	groupNames, err = getAllAutoScalingGroups(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Auto Scaling Groups")
	}

	assert.Contains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}

func TestNukeAutoScalingGroups(t *testing.T) {
	t.Parallel()

	region := getRandomRegion()
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := autoscaling.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + util.UniqueID()
	createTestAutoScalingGroup(t, session, uniqueTestID)

	// clean up ec2 instance created by the above call
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

	_, err = svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&uniqueTestID},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllAutoScalingGroups(session, []*string{&uniqueTestID}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	groupNames, err := getAllAutoScalingGroups(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Auto Scaling Groups")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}
