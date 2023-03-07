package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// Test helpers
func createElasticBeanstalkEnvironment(t *testing.T, region string) (string, error) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	ebsService := elasticbeanstalk.New(session)

	environmentName := strings.ToLower(fmt.Sprintf("cloud-nuke-test-%s-%s", util.UniqueID(), util.UniqueID()))

	testApplicationName, createAppErr := createElasticBeanstalkApplication(t, region)
	require.NoError(t, createAppErr)

	param := &elasticbeanstalk.CreateEnvironmentInput{
		ApplicationName: aws.String(testApplicationName),
		// See: https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/concepts-roles-service.html
		OperationsRole:  aws.String(formatEBServiceRoleName(t)),
		Description:     aws.String("Test environment created by cloud-nuke - It is safe to delete me!"),
		EnvironmentName: aws.String(environmentName),
		// See: https://docs.aws.amazon.com/elasticbeanstalk/latest/platforms/platforms-supported.html
		// For the list of currently supported "Platforms" (i.e. SolutionStackNames)
		SolutionStackName: aws.String("64bit Amazon Linux 2 v3.6.4 running Go 1"),
		Tags:              []*elasticbeanstalk.Tag{},
		Tier:              &elasticbeanstalk.EnvironmentTier{},
		OptionSettings: []*elasticbeanstalk.ConfigurationOptionSetting{
			// See https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/iam-instanceprofile.html
			// When you create an Elastic Beanstalk environment via the CLI or the API, AWS creates a default
			// instance profile named `aws-elasticbeanstalk-ec2-role` and assigns managed policies with default permissions to it
			// While that's all well and good, environment creation still fails if you do not supply this ConfigurationOptionSetting
			{
				Namespace:  aws.String("aws:autoscaling:launchconfiguration"),
				OptionName: aws.String("IamInstanceProfile"),
				Value:      aws.String("aws-elasticbeanstalk-ec2-role"),
			},
		},
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

// Test helpers

// Convenience method to create an Elastic Beanstalk environment in a random region
func setupEbEnvironmentTest(t *testing.T) (string, *session.Session) {
	t.Helper()

	// TODO - revert me
	region := "us-east-1"
	/*
		region, err := getRandomRegion()
		require.NoError(t, err)
	*/

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	return region, session
}

// Convenience method to format the ARN for the Elastic Beanstalk service role in a manner that portable across AWS accounts
func formatEBServiceRoleName(t *testing.T) string {
	t.Helper()

	stsService := sts.New(session.New(&aws.Config{}))

	output, err := stsService.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	require.NoError(t, err)

	templateString := "arn:aws:iam::%s:role/aws-elasticbeanstalk-service-role"

	return fmt.Sprintf(templateString, aws.StringValue(output.Account))
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

	// Wait on the deletion of the Elastic Beanstalk environment, so that we don't accidentally return too early
	waitErr := ebsService.WaitUntilEnvironmentTerminated(&elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName: environmentName,
	})
	require.NoError(t, waitErr)

	_, deleteEnvironmentErr := ebsService.TerminateEnvironment(param)
	require.NoError(t, deleteEnvironmentErr)
}

func assertElasticBeanstalkEnvironmentDeleted(t *testing.T, session *session.Session, environmentName string) {
	time.Sleep(30 * time.Second)
	environments, err := getAllElasticBeanstalkEnvironments(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, environments, environmentName)
}
