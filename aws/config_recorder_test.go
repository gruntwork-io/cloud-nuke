package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListConfigRecorders(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	// You can only have one configuration recorder per region, so instead of our usual
	// create and list pattern, we'll just ensure there is a recorder in our target region,
	// creating one if necessary, and then that we can see that config recorder returned by
	// getAllConfigRecorders
	configRecorderName := ensureConfigurationRecorderExistsInRegion(t, region)

	configRecorderNames, lookupErr := getAllConfigRecorders(session, time.Now(), config.Config{})
	require.NoError(t, lookupErr)
	require.NotEmpty(t, configRecorderNames)

	// Sanity check that we got back a recorder
	assert.Equal(t, configRecorderNames[0], configRecorderName)
}

func TestNukeConfigRecorderOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configRecorderName := ensureConfigurationRecorderExistsInRegion(t, region)

	defer deleteConfigRecorder(t, region, configRecorderName, false)

	require.NoError(
		t,
		nukeAllConfigRecorders(session, []string{configRecorderName}),
	)

	assertConfigRecordersDeleted(t, region)
}

// Test helpers

func deleteConfigRecorder(t *testing.T, region string, configRecorderName string, checkErr bool) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configService := configservice.New(session)

	param := &configservice.DeleteConfigurationRecorderInput{
		ConfigurationRecorderName: aws.String(configRecorderName),
	}

	_, deleteErr := configService.DeleteConfigurationRecorder(param)
	if checkErr {
		require.NoError(t, deleteErr)
	}
}

func assertConfigRecordersDeleted(t *testing.T, region string) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := configservice.New(session)

	param := &configservice.DescribeConfigurationRecordersInput{}

	resp, err := svc.DescribeConfigurationRecorders(param)
	require.NoError(t, err)

	require.Empty(t, resp.ConfigurationRecorders)
}
