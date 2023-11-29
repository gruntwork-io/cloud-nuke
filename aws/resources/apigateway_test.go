package resources

import (
	"context"
	"testing"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/andrewderr/cloud-nuke-a1/util"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedApiGateway struct {
	apigatewayiface.APIGatewayAPI
	GetRestApisResp   apigateway.GetRestApisOutput
	DeleteRestApiResp apigateway.DeleteRestApiOutput
}

func (m mockedApiGateway) GetRestApis(*apigateway.GetRestApisInput) (*apigateway.GetRestApisOutput, error) {
	// Only need to return mocked response output
	return &m.GetRestApisResp, nil
}

func (m mockedApiGateway) DeleteRestApi(*apigateway.DeleteRestApiInput) (*apigateway.DeleteRestApiOutput, error) {
	// Only need to return mocked response output
	return &m.DeleteRestApiResp, nil
}

func TestAPIGatewayGetAllAndNukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testApiID := "aws-nuke-test-" + util.UniqueID()
	apiGateway := ApiGateway{
		Client: mockedApiGateway{
			GetRestApisResp: apigateway.GetRestApisOutput{
				Items: []*apigateway.RestApi{
					{Id: aws.String(testApiID)},
				},
			},
			DeleteRestApiResp: apigateway.DeleteRestApiOutput{},
		},
	}

	apis, err := apiGateway.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.Contains(t, awsgo.StringValueSlice(apis), testApiID)

	err = apiGateway.nukeAll([]*string{aws.String(testApiID)})
	require.NoError(t, err)
}

func TestAPIGatewayGetAllTimeFilter(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testApiID := "aws-nuke-test-" + util.UniqueID()
	now := time.Now()
	apiGateway := ApiGateway{
		Client: mockedApiGateway{
			GetRestApisResp: apigateway.GetRestApisOutput{
				Items: []*apigateway.RestApi{{
					Id:          aws.String(testApiID),
					CreatedDate: aws.Time(now),
				}},
			},
		},
	}

	// test API is not excluded from the filter
	IDs, err := apiGateway.getAll(context.Background(), config.Config{
		APIGateway: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(1)),
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(IDs), testApiID)

	// test API being excluded from the filter
	apiGwIdsOlder, err := apiGateway.getAll(context.Background(), config.Config{
		APIGateway: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-1)),
			},
		},
	})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(apiGwIdsOlder), testApiID)
}

func TestNukeAPIGatewayMoreThanOne(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testApiID1 := "aws-nuke-test-" + util.UniqueID()
	testApiID2 := "aws-nuke-test-" + util.UniqueID()
	apiGateway := ApiGateway{
		Client: mockedApiGateway{
			GetRestApisResp: apigateway.GetRestApisOutput{
				Items: []*apigateway.RestApi{
					{Id: aws.String(testApiID1)},
					{Id: aws.String(testApiID2)},
				},
			},
			DeleteRestApiResp: apigateway.DeleteRestApiOutput{},
		},
	}

	apis, err := apiGateway.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.Contains(t, awsgo.StringValueSlice(apis), testApiID1)
	require.Contains(t, awsgo.StringValueSlice(apis), testApiID2)

	err = apiGateway.nukeAll([]*string{aws.String(testApiID1), aws.String(testApiID2)})
	require.NoError(t, err)
}
