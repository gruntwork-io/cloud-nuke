package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestLaunchConfiguration(t *testing.T, session *session.Session, name string) {
	svc := autoscaling.New(session)
	instance, err := CreateTestEC2Instance(session, name, false)
	if err != nil {
		assert.Failf(t, "Could not create test EC2 instance", errors.WithStackTrace(err).Error())
	}

	param := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: &name,
		InstanceId:              instance.InstanceId,
	}

	_, err = svc.CreateLaunchConfiguration(param)
	if err != nil {
		assert.Failf(t, "Could not create test Launch Configuration", errors.WithStackTrace(err).Error())
	}

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
}

func TestListLaunchConfigurations(t *testing.T) {
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
	createTestLaunchConfiguration(t, session, uniqueTestID)

	// clean up after this test
	defer nukeAllLaunchConfigurations(session, []*string{&uniqueTestID})
	instances, err := findEC2InstancesByNameTag(session, uniqueTestID)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	defer nukeAllEc2Instances(session, instances, true)

	configNames, err := getAllLaunchConfigurations(session, region, time.Now().Add(1*time.Hour*-1))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Configurations")
	}

	assert.NotContains(t, awsgo.StringValueSlice(configNames), uniqueTestID)

	configNames, err = getAllLaunchConfigurations(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Configurations")
	}

	assert.Contains(t, awsgo.StringValueSlice(configNames), uniqueTestID)
}

func TestNukeLaunchConfigurations(t *testing.T) {
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
	createTestLaunchConfiguration(t, session, uniqueTestID)

	// clean up ec2 instance created by the above call
	instances, err := findEC2InstancesByNameTag(session, uniqueTestID)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	defer nukeAllEc2Instances(session, instances, true)

	_, err = svc.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{&uniqueTestID},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllLaunchConfigurations(session, []*string{&uniqueTestID}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	groupNames, err := getAllLaunchConfigurations(session, region, time.Now().Add(1*time.Hour))
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Configurations")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}
