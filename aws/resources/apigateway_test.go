package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedApiGateway struct {
	ApiGatewayAPI
	GetRestApisOutput             apigateway.GetRestApisOutput
	GetStagesOutput               apigateway.GetStagesOutput
	DeleteClientCertificateOutput apigateway.DeleteClientCertificateOutput
	DeleteRestApiOutput           apigateway.DeleteRestApiOutput

	GetDomainNamesOutput        apigateway.GetDomainNamesOutput
	GetBasePathMappingsOutput   apigateway.GetBasePathMappingsOutput
	DeleteBasePathMappingOutput apigateway.DeleteBasePathMappingOutput
}

func (m mockedApiGateway) GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error) {
	return &m.GetRestApisOutput, nil
}
func (m mockedApiGateway) GetStages(ctx context.Context, params *apigateway.GetStagesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetStagesOutput, error) {
	return &m.GetStagesOutput, nil
}

func (m mockedApiGateway) DeleteClientCertificate(ctx context.Context, params *apigateway.DeleteClientCertificateInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteClientCertificateOutput, error) {
	return &m.DeleteClientCertificateOutput, nil
}

func (m mockedApiGateway) DeleteRestApi(ctx context.Context, params *apigateway.DeleteRestApiInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteRestApiOutput, error) {
	return &m.DeleteRestApiOutput, nil
}

func (m mockedApiGateway) GetDomainNames(ctx context.Context, params *apigateway.GetDomainNamesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetDomainNamesOutput, error) {
	return &m.GetDomainNamesOutput, nil
}

func (m mockedApiGateway) GetBasePathMappings(ctx context.Context, params *apigateway.GetBasePathMappingsInput, optFns ...func(*apigateway.Options)) (*apigateway.GetBasePathMappingsOutput, error) {
	return &m.GetBasePathMappingsOutput, nil
}

func (m mockedApiGateway) DeleteBasePathMapping(ctx context.Context, params *apigateway.DeleteBasePathMappingInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteBasePathMappingOutput, error) {
	return &m.DeleteBasePathMappingOutput, nil
}

func TestAPIGatewayGetAllAndNukeAll(t *testing.T) {
	t.Parallel()

	testApiID := "aws-nuke-test-" + util.UniqueID()
	mockClient := mockedApiGateway{
		GetRestApisOutput: apigateway.GetRestApisOutput{
			Items: []types.RestApi{
				{Id: aws.String(testApiID)},
			},
		},
		DeleteRestApiOutput: apigateway.DeleteRestApiOutput{},
	}

	apis, err := listApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(apis), testApiID)

	err = deleteApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, "apigateway", []*string{aws.String(testApiID)})
	require.NoError(t, err)
}

func TestAPIGatewayGetAllTimeFilter(t *testing.T) {
	t.Parallel()

	testApiID := "aws-nuke-test-" + util.UniqueID()
	now := time.Now()
	mockClient := mockedApiGateway{
		GetRestApisOutput: apigateway.GetRestApisOutput{
			Items: []types.RestApi{{
				Id:          aws.String(testApiID),
				CreatedDate: aws.Time(now),
			}},
		},
	}

	// test API is not excluded from the filter
	IDs, err := listApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(1)),
		},
	})
	require.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(IDs), testApiID)

	// test API being excluded from the filter
	apiGwIdsOlder, err := listApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-1)),
		},
	})
	require.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(apiGwIdsOlder), testApiID)
}

func TestNukeAPIGatewayMoreThanOne(t *testing.T) {
	t.Parallel()

	testApiID1 := "aws-nuke-test-" + util.UniqueID()
	testApiID2 := "aws-nuke-test-" + util.UniqueID()
	mockClient := mockedApiGateway{
		GetRestApisOutput: apigateway.GetRestApisOutput{
			Items: []types.RestApi{
				{Id: aws.String(testApiID1)},
				{Id: aws.String(testApiID2)},
			},
		},
		DeleteRestApiOutput: apigateway.DeleteRestApiOutput{},
	}

	apis, err := listApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(apis), testApiID1)
	require.Contains(t, aws.ToStringSlice(apis), testApiID2)

	err = deleteApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, "apigateway", []*string{aws.String(testApiID1), aws.String(testApiID2)})
	require.NoError(t, err)
}

func TestNukeAPIGatewayWithCertificates(t *testing.T) {
	t.Parallel()

	testApiID1 := "aws-nuke-test-" + util.UniqueID()
	testApiID2 := "aws-nuke-test-" + util.UniqueID()

	clientCertID := "aws-client-cert" + util.UniqueID()
	mockClient := mockedApiGateway{
		GetRestApisOutput: apigateway.GetRestApisOutput{
			Items: []types.RestApi{
				{Id: aws.String(testApiID1)},
				{Id: aws.String(testApiID2)},
			},
		},
		GetStagesOutput: apigateway.GetStagesOutput{
			Item: []types.Stage{
				{
					ClientCertificateId: aws.String(clientCertID),
				},
			},
		},
		DeleteClientCertificateOutput: apigateway.DeleteClientCertificateOutput{},
		DeleteRestApiOutput:           apigateway.DeleteRestApiOutput{},
	}

	apis, err := listApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(apis), testApiID1)
	require.Contains(t, aws.ToStringSlice(apis), testApiID2)

	err = deleteApiGateways(context.Background(), mockClient, resource.Scope{Region: "us-east-1"}, "apigateway", []*string{aws.String(testApiID1), aws.String(testApiID2)})
	require.NoError(t, err)
}

func TestDeleteAssociatedApiMappingsV1(t *testing.T) {
	t.Parallel()

	apiIDToDelete := "test-api-id"
	basePath := "test"
	domainName := "test.example.com"

	mockClient := &mockedApiGateway{
		GetDomainNamesOutput: apigateway.GetDomainNamesOutput{
			Items: []types.DomainName{
				{DomainName: aws.String(domainName)},
			},
		},
		GetBasePathMappingsOutput: apigateway.GetBasePathMappingsOutput{
			Items: []types.BasePathMapping{
				{
					BasePath:  aws.String(basePath),
					RestApiId: aws.String(apiIDToDelete),
					Stage:     aws.String("prod"),
				},
				{
					BasePath:  aws.String("unrelated"),
					RestApiId: aws.String("some-other-api"),
				},
			},
		},
	}

	err := deleteAssociatedApiMappingsV1(context.Background(), mockClient, []*string{aws.String(apiIDToDelete)})
	require.NoError(t, err)
}
