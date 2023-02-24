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

func setupEbEnvironmentTest(t *testing.T) (string, *session.Session) {
	t.Helper()
	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	return region, session
}

func TestListElasticBeanstalkEnvironments(t *testing.T) {
	t.Parallel()

	region, session := setupEbEnvironmentTest(t)

	environmentName, createEnvironmentErr := createElasticBeanstalkEnvironment(t, region)
	require.NoError(t, createEnvironmentErr)

	defer deleteElasticBeanstalkEnvironment(t, aws.String(region), aws.String(environmentName), false)

	environments, err := getAllElasticBeanstalkEnvironments(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, environments, environmentName)
}

func TestNukeElasticBeanstalkEnvironmentOne(t *testing.T) {
	t.Parallel()

	region, session := setupEbEnvironmentTest(t)

	environmentName, createEnvironmentErr := createElasticBeanstalkEnvironment(t, region)
	require.NoError(t, createEnvironmentErr)

	defer deleteElasticBeanstalkEnvironment(t, aws.String(region), aws.String(environmentName), false)

	identifiers := []string{environmentName}

	require.NoError(t, nukeAllElasticBeanstalkEnvironments(session, aws.StringSlice(identifiers)))
	assertElasticBeanstalkEnvironmentDeleted(t, session, environmentName)
}

func TestNukeElasticBeanstalkEnvironmentMultiple(t *testing.T) {
	t.Parallel()

	region, session := setupEbEnvironmentTest(t)

	environmentName1, createEnvironmentErr1 := createElasticBeanstalkEnvironment(t, region)
	require.NoError(t, createEnvironmentErr1)

	environmentName2, createEnvironmentErr2 := createElasticBeanstalkEnvironment(t, region)
	require.NoError(t, createEnvironmentErr2)

	identifiers := []string{environmentName1, environmentName2}

	require.NoError(t, nukeAllElasticBeanstalkEnvironments(session, aws.StringSlice(identifiers)))
	assertElasticBeanstalkEnvironmentDeleted(t, session, environmentName1)
	assertElasticBeanstalkEnvironmentDeleted(t, session, environmentName2)
}

// Test helpers
func createElasticBeanstalkEnvironment(t *testing.T, region string) (string, error) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	ebsService := elasticbeanstalk.New(session)

	name := strings.ToLower(fmt.Sprintf("cloud-nuke-test-%s-%s", util.UniqueID(), util.UniqueID()))

	testApplicationName, createAppErr := createElasticBeanstalkApplication(t, region)
	require.NoError(t, createAppErr)

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

	// Wait on the new Elastic Beanstalk environment to come up
	waitErr := ebsService.WaitUntilEnvironmentExists(&elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName: aws.String(testApplicationName),
	})
	require.NoError(t, waitErr)

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

func assertElasticBeanstalkEnvironmentDeleted(t *testing.T, session *session.Session, environmentName string) {
	environments, err := getAllElasticBeanstalkEnvironments(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, environments, environmentName)
}
