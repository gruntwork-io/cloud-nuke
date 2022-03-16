package aws

import (
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
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
		assert.Failf(t, "Could not create test ASG", errors.WithStackTrace(err).Error())
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

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	createTestAutoScalingGroup(t, session, uniqueTestID)
	// clean up after this test
	defer nukeAllAutoScalingGroups(session, []*string{&uniqueTestID})
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

	groupNames, err := getAllAutoScalingGroups(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Auto Scaling Groups")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)

	groupNames, err = getAllAutoScalingGroups(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Auto Scaling Groups")
	}

	assert.Contains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}

func TestNukeAutoScalingGroups(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := autoscaling.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
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

	groupNames, err := getAllAutoScalingGroups(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Auto Scaling Groups")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}

// Test config file filtering works as expected
func TestShouldIncludeAutoScalingGroup(t *testing.T) {
	mockAutoScalingGroup := &autoscaling.Group{
		AutoScalingGroupName: awsgo.String("cloud-nuke-test"),
		CreatedTime:          awsgo.Time(time.Now()),
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		logging.Logger.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		AutoScalingGroup: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	mockIncludeConfig := config.Config{
		AutoScalingGroup: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	cases := []struct {
		Name             string
		AutoScalingGroup *autoscaling.Group
		Config           config.Config
		ExcludeAfter     time.Time
		Expected         bool
	}{
		{
			Name:             "ConfigExclude",
			AutoScalingGroup: mockAutoScalingGroup,
			Config:           mockExcludeConfig,
			ExcludeAfter:     time.Now().Add(1 * time.Hour),
			Expected:         false,
		},
		{
			Name:             "ConfigInclude",
			AutoScalingGroup: mockAutoScalingGroup,
			Config:           mockIncludeConfig,
			ExcludeAfter:     time.Now().Add(1 * time.Hour),
			Expected:         true,
		},
		{
			Name:             "NotOlderThan",
			AutoScalingGroup: mockAutoScalingGroup,
			Config:           config.Config{},
			ExcludeAfter:     time.Now().Add(1 * time.Hour * -1),
			Expected:         false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeAutoScalingGroup(c.AutoScalingGroup, c.ExcludeAfter, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}
