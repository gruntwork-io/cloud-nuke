package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
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

func createTestLaunchConfiguration(t *testing.T, session *session.Session, name string) {
	svc := autoscaling.New(session)
	instance := createTestEC2Instance(t, session, name, false)

	param := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: &name,
		InstanceId:              instance.InstanceId,
	}

	_, err := svc.CreateLaunchConfiguration(param)
	if err != nil {
		assert.Failf(t, "Could not create test Launch Configuration", errors.WithStackTrace(err).Error())
	}

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
}

func TestListLaunchConfigurations(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
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
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

	configNames, err := getAllLaunchConfigurations(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Configurations")
	}

	assert.NotContains(t, awsgo.StringValueSlice(configNames), uniqueTestID)

	configNames, err = getAllLaunchConfigurations(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Configurations")
	}

	assert.Contains(t, awsgo.StringValueSlice(configNames), uniqueTestID)
}

func TestNukeLaunchConfigurations(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
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
	defer nukeAllEc2Instances(session, findEC2InstancesByNameTag(t, session, uniqueTestID))

	_, err = svc.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{&uniqueTestID},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllLaunchConfigurations(session, []*string{&uniqueTestID}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	groupNames, err := getAllLaunchConfigurations(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Configurations")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}

func TestShouldIncludeLaunchConfiguration(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	mockLaunchConfiguration := &autoscaling.LaunchConfiguration{
		LaunchConfigurationName: awsgo.String("cloud-nuke-test"),
		CreatedTime:             awsgo.Time(time.Now()),
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		logging.Logger.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		LaunchConfiguration: config.ResourceType{
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
		LaunchConfiguration: config.ResourceType{
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
		Name                string
		LaunchConfiguration *autoscaling.LaunchConfiguration
		Config              config.Config
		ExcludeAfter        time.Time
		Expected            bool
	}{
		{
			Name:                "ConfigExclude",
			LaunchConfiguration: mockLaunchConfiguration,
			Config:              mockExcludeConfig,
			ExcludeAfter:        time.Now().Add(1 * time.Hour),
			Expected:            false,
		},
		{
			Name:                "ConfigInclude",
			LaunchConfiguration: mockLaunchConfiguration,
			Config:              mockIncludeConfig,
			ExcludeAfter:        time.Now().Add(1 * time.Hour),
			Expected:            true,
		},
		{
			Name:                "NotOlderThan",
			LaunchConfiguration: mockLaunchConfiguration,
			Config:              config.Config{},
			ExcludeAfter:        time.Now().Add(1 * time.Hour * -1),
			Expected:            false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeLaunchConfiguration(c.LaunchConfiguration, c.ExcludeAfter, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}
