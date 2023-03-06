package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestAPIGateway struct {
	ID   *string
	Name *string
}

func createTestAPIGateway(t *testing.T, session *session.Session, name string) (*TestAPIGateway, error) {
	svc := apigateway.New(session)

	testGw := &TestAPIGateway{
		Name: aws.String(name),
	}

	param := &apigateway.CreateRestApiInput{
		Name: aws.String(name),
	}

	output, err := svc.CreateRestApi(param)
	if err != nil {
		assert.Failf(t, "Could not create test API Gateway: %s", errors.WithStackTrace(err).Error())
	}

	testGw.ID = output.Id

	return testGw, nil
}

func TestListAPIGateways(t *testing.T) {
	t.Parallel()
	session, err := getAwsSession(false)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	apigwName := "aws-nuke-test-" + util.UniqueID()
	testGw, createTestGwErr := createTestAPIGateway(t, session, apigwName)
	require.NoError(t, createTestGwErr)
	// clean up after this test
	defer nukeAllAPIGateways(session, []*string{testGw.ID})

	apigwIds, err := getAllAPIGateways(session, time.Now(), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of API Gateways (v1)")
	}

	assert.Contains(t, awsgo.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))
}

func TestTimeFilterExclusionNewlyCreatedAPIGateway(t *testing.T) {
	t.Parallel()

	session, err := getAwsSession(false)
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()

	testGw, createTestGwErr := createTestAPIGateway(t, session, apigwName)
	require.NoError(t, createTestGwErr)
	defer nukeAllAPIGateways(session, []*string{testGw.ID})

	// Assert API Gateway is picked up without filters
	apigwIds, err := getAllAPIGateways(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))

	// Assert API Gateway doesn't appear when we look at API Gateways older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	apiGwIdsOlder, err := getAllAPIGateways(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(apiGwIdsOlder), aws.StringValue(testGw.ID))
}

func TestNukeAPIGatewayOne(t *testing.T) {
	t.Parallel()

	session, err := getAwsSession(false)
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testGw, createTestErr := createTestAPIGateway(t, session, apigwName)
	require.NoError(t, createTestErr)

	nukeErr := nukeAllAPIGateways(session, []*string{testGw.ID})
	require.NoError(t, nukeErr)

	// Make sure the API Gateway was deleted
	apigwIds, err := getAllAPIGateways(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))
}

func TestNukeAPIGatewayMoreThanOne(t *testing.T) {
	t.Parallel()

	session, err := getAwsSession(false)
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()
	apigwName2 := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testGw, createTestErr := createTestAPIGateway(t, session, apigwName)
	require.NoError(t, createTestErr)
	testGw2, createTestErr2 := createTestAPIGateway(t, session, apigwName2)
	require.NoError(t, createTestErr2)

	nukeErr := nukeAllAPIGateways(session, []*string{testGw.ID, testGw2.ID})
	require.NoError(t, nukeErr)

	// Make sure the API Gateway was deleted
	apigwIds, err := getAllAPIGateways(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))
	assert.NotContains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw2.ID))
}
