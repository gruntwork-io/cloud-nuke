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

func setupEbApplicationTest(t *testing.T) (string, *session.Session) {
	t.Helper()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	return region, session
}

func TestListElasticBeanstalkApplications(t *testing.T) {
	t.Parallel()

	region, session := setupEbApplicationTest(t)

	applicationName, createApplicationErr := createElasticBeanstalkApplication(t, region)
	require.NoError(t, createApplicationErr)

	defer deleteElasticBeanstalkApplication(t, aws.String(region), aws.String(applicationName), false)

	applications, err := getAllElasticBeanstalkApplications(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, applications, applicationName)
}

func TestNukeElasticBeanstalkApplicationOne(t *testing.T) {
	t.Parallel()

	region, session := setupEbApplicationTest(t)

	applicationName, createApplicationErr := createElasticBeanstalkApplication(t, region)
	require.NoError(t, createApplicationErr)

	defer deleteElasticBeanstalkApplication(t, aws.String(region), aws.String(applicationName), true)

	identifiers := []string{applicationName}

	require.NoError(t, nukeAllElasticBeanstalkApplications(session, aws.StringSlice(identifiers)))

	// Allow for propagation
	time.Sleep(15 * time.Second)

	assertElasticBeanstalkApplicationDeleted(t, session, applicationName)
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

func deleteElasticBeanstalkApplication(t *testing.T, region *string, applicationName *string, force bool) {
	session, err := session.NewSession(&aws.Config{Region: region})
	require.NoError(t, err)

	ebsService := elasticbeanstalk.New(session)

	input := &elasticbeanstalk.DeleteApplicationInput{
		ApplicationName:     applicationName,
		TerminateEnvByForce: aws.Bool(force),
	}

	_, err = ebsService.DeleteApplication(input)
	require.NoError(t, err)
}

func assertElasticBeanstalkApplicationDeleted(t *testing.T, session *session.Session, applicationName string) {
	applications, err := getAllElasticBeanstalkApplications(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, applications, applicationName)
}
