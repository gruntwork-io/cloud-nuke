package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestAPIGatewayV2 struct {
	ID   *string
	Name *string
}

func createTestAPIGatewayV2(t *testing.T, session *session.Session, name string) (*TestAPIGatewayV2, error) {
	svc := apigatewayv2.New(session)

	testGw := &TestAPIGatewayV2{
		Name: aws.String(name),
	}

	param := &apigatewayv2.CreateApiInput{
		Name:         aws.String(name),
		ProtocolType: aws.String("HTTP"),
	}

	output, err := svc.CreateApi(param)
	if err != nil {
		assert.Failf(t, "Could not create test API Gateway (v2): %s", errors.WithStackTrace(err).Error())
	}

	testGw.ID = output.ApiId
	return testGw, nil
}

func TestListAPIGatewaysV2(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
	)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	apigwName := "aws-nuke-test-" + util.UniqueID()
	testGw, createTestGwErr := createTestAPIGatewayV2(t, session, apigwName)
	require.NoError(t, createTestGwErr)
	// clean up after this test
	defer nukeAllAPIGatewaysV2(session, []*string{testGw.ID})

	apigwIds, err := getAllAPIGatewaysV2(session, time.Now(), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of API Gateways (v2)")
	}

	assert.Contains(t, awsgo.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))
}

func TestTimeFilterExclusionNewlyCreatedAPIGatewayV2(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()

	testGw, createTestGwErr := createTestAPIGatewayV2(t, session, apigwName)
	require.NoError(t, createTestGwErr)
	defer nukeAllAPIGatewaysV2(session, []*string{testGw.ID})

	// Assert API Gateway is picked up without filters
	apigwIds, err := getAllAPIGatewaysV2(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))

	// Assert API Gateway doesn't appear when we look at API Gateways older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	apiGwIdsOlder, err := getAllAPIGatewaysV2(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(apiGwIdsOlder), aws.StringValue(testGw.ID))
}

func TestNukeAPIGatewayV2One(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testGw, createTestErr := createTestAPIGatewayV2(t, session, apigwName)
	require.NoError(t, createTestErr)

	nukeErr := nukeAllAPIGatewaysV2(session, []*string{testGw.ID})
	require.NoError(t, nukeErr)

	// Make sure the API Gateway was deleted
	apigwIds, err := getAllAPIGatewaysV2(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))
}

func TestNukeAPIGatewayV2MoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	apigwName := "aws-nuke-test-" + util.UniqueID()
	apigwName2 := "aws-nuke-test-" + util.UniqueID()
	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	testGw, createTestErr := createTestAPIGatewayV2(t, session, apigwName)
	require.NoError(t, createTestErr)
	testGw2, createTestErr2 := createTestAPIGatewayV2(t, session, apigwName2)
	require.NoError(t, createTestErr2)

	nukeErr := nukeAllAPIGatewaysV2(session, []*string{testGw.ID, testGw2.ID})
	require.NoError(t, nukeErr)

	// Make sure the API Gateway was deleted
	apigwIds, err := getAllAPIGatewaysV2(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw.ID))
	assert.NotContains(t, aws.StringValueSlice(apigwIds), aws.StringValue(testGw2.ID))
}
