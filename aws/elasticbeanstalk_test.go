package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListElasticBeanstalkEnvironments(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	environmentName, createEnvironmentErr := createElasticBeanstalkEnvironment(t, region)
	require.NoError(t, createEnvironmentErr)

	defer deleteElasticBeanstalkEnvironment(t, aws.String(region), aws.String(environmentName), false)

	environments, err := getAllElasticBeanstalkEnvironments(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, environments, environmentName)
}

// Test helpers

func createElasticBeanstalkApplication(t *testing.T, region string) (string, error) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	ebsService := elasticbeanstalk.New(session)

	applicationName := strings.ToLower(fmt.Sprintf("cloud-nuke-test-app-%s-%s", util.UniqueID(), util.UniqueID()))

	input := &elasticbeanstalk.CreateApplicationInput{
		ApplicationName: aws.String(applicationName),
		Description:     aws.String("Test application created by cloud-nuke - probably safe to delete"),
	}

	appDescriptionMsg, err := ebsService.CreateApplication(input)
	require.NoError(t, err)

	return aws.StringValue(appDescriptionMsg.Application.ApplicationName), nil
}

func createElasticBeanstalkEnvironment(t *testing.T, region string) (string, error) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	ebsService := elasticbeanstalk.New(session)

	name := strings.ToLower(fmt.Sprintf("cloud-nuke-test-%s-%s", util.UniqueID(), util.UniqueID()))

	testApplicationName, creatAppErr := createElasticBeanstalkApplication(t, region)
	require.NoError(t, creatAppErr)

	// Wait on the new Elastic Beanstalk environment to come up
	waitErr := ebsService.WaitUntilEnvironmentExists(&elasticbeanstalk.DescribeEnvironmentsInput{
		EnvironmentNames: []*string{aws.String(name)},
	})
	require.NoError(t, waitErr)

	param := &elasticbeanstalk.CreateEnvironmentInput{
		ApplicationName:   aws.String(testApplicationName),
		Description:       aws.String("Test environment created by cloud-nuke - probably safe to delete"),
		EnvironmentName:   aws.String(name),
		SolutionStackName: aws.String("64bit Debian jessie v2.16.0 running Go 1.4 (Preconfigured - Docker)"),
		Tags:              []*elasticbeanstalk.Tag{},
		Tier:              &elasticbeanstalk.EnvironmentTier{},
	}

	output, createEnvironmentErr := ebsService.CreateEnvironment(param)
	require.NoError(t, createEnvironmentErr)

	return aws.StringValue(output.EnvironmentName), nil
}

func deleteElasticBeanstalkEnvironment(t *testing.T, region *string, environmentName *string, checkErr bool) {
	session, err := session.NewSession(&aws.Config{Region: region})
	require.NoError(t, err)

	ebsService := elasticbeanstalk.New(session)

	param := &elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentName:    environmentName,
		ForceTerminate:     aws.Bool(true),
		TerminateResources: aws.Bool(true),
	}

	_, deleteEnvironmentErr := ebsService.TerminateEnvironment(param)
	require.NoError(t, deleteEnvironmentErr)
}
