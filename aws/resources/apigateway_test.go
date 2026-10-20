package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedApiGateway struct {
	ApiGatewayAPI
	GetRestApisOutput   apigateway.GetRestApisOutput
	DeleteRestApiOutput apigateway.DeleteRestApiOutput
	// DomainPages are returned by page token.
	DomainPages       []apigateway.GetDomainNamesOutput
	GetDomainNamesErr error
	// MappingPages are keyed by domain name and returned by page token.
	MappingPages map[string][]apigateway.GetBasePathMappingsOutput
	// GetBasePathMappingsErr fails the listing on the domains it names.
	GetBasePathMappingsErr   map[string]error
	DeleteBasePathMappingErr error
	Calls                    *apiGatewayCallLog
}

func (m mockedApiGateway) GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error) {
	return &m.GetRestApisOutput, nil
}

func (m mockedApiGateway) DeleteRestApi(ctx context.Context, params *apigateway.DeleteRestApiInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteRestApiOutput, error) {
	m.Calls.add("DeleteRestApi " + aws.ToString(params.RestApiId))
	return &m.DeleteRestApiOutput, nil
}

func (m mockedApiGateway) GetDomainNames(ctx context.Context, params *apigateway.GetDomainNamesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetDomainNamesOutput, error) {
	m.Calls.add("GetDomainNames")
	if m.GetDomainNamesErr != nil {
		return nil, m.GetDomainNamesErr
	}
	if len(m.DomainPages) == 0 {
		return &apigateway.GetDomainNamesOutput{}, nil
	}
	return &m.DomainPages[apiGatewayPage(params.Position)], nil
}

func (m mockedApiGateway) GetBasePathMappings(ctx context.Context, params *apigateway.GetBasePathMappingsInput, optFns ...func(*apigateway.Options)) (*apigateway.GetBasePathMappingsOutput, error) {
	m.Calls.add("GetBasePathMappings " + aws.ToString(params.DomainName) + " id=" + aws.ToString(params.DomainNameId))
	if err := m.GetBasePathMappingsErr[aws.ToString(params.DomainName)]; err != nil {
		return nil, err
	}
	pages := m.MappingPages[aws.ToString(params.DomainName)]
	if len(pages) == 0 {
		return &apigateway.GetBasePathMappingsOutput{}, nil
	}
	return &pages[apiGatewayPage(params.Position)], nil
}

func (m mockedApiGateway) DeleteBasePathMapping(ctx context.Context, params *apigateway.DeleteBasePathMappingInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteBasePathMappingOutput, error) {
	if m.DeleteBasePathMappingErr != nil {
		return nil, m.DeleteBasePathMappingErr
	}
	m.Calls.add("DeleteBasePathMapping " + aws.ToString(params.DomainName) + " id=" + aws.ToString(params.DomainNameId) + " " + aws.ToString(params.BasePath))
	return &apigateway.DeleteBasePathMappingOutput{}, nil
}

func TestAPIGateway_GetAll(t *testing.T) {
	t.Parallel()

	testApiID1 := "api-1"
	testApiID2 := "api-2"
	testApiName1 := "test-api-1"
	testApiName2 := "test-api-2"
	now := time.Now()

	mock := mockedApiGateway{
		GetRestApisOutput: apigateway.GetRestApisOutput{
			Items: []types.RestApi{
				{
					Id:          aws.String(testApiID1),
					Name:        aws.String(testApiName1),
					CreatedDate: aws.Time(now),
					Tags:        map[string]string{"env": "dev"},
				},
				{
					Id:          aws.String(testApiID2),
					Name:        aws.String(testApiName2),
					CreatedDate: aws.Time(now.Add(1 * time.Hour)),
					Tags:        map[string]string{"env": "prod"},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testApiID1, testApiID2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test-api-1"),
					}},
				},
			},
			expected: []string{testApiID2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testApiID1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testApiID1},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testApiID2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			apis, err := listApiGateways(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(apis))
		})
	}
}

func TestAPIGateway_NukeAll(t *testing.T) {
	t.Parallel()

	// Two domains on two pages, the second private; api-1 is mapped on both, the second time on the second page of
	// mappings.
	domains := []apigateway.GetDomainNamesOutput{
		{Items: []types.DomainName{{DomainName: aws.String("a.example.com")}}, Position: aws.String("1")},
		{Items: []types.DomainName{{DomainName: aws.String("b.example.com"), DomainNameId: aws.String("abc123")}}},
	}
	mappings := map[string][]apigateway.GetBasePathMappingsOutput{
		"a.example.com": {
			{Items: []types.BasePathMapping{
				{RestApiId: aws.String("api-1"), BasePath: aws.String("(none)")},
				{RestApiId: aws.String("api-2"), BasePath: aws.String("v2")},
			}},
		},
		"b.example.com": {
			{Items: []types.BasePathMapping{{RestApiId: aws.String("api-2"), BasePath: aws.String("other")}}, Position: aws.String("1")},
			{Items: []types.BasePathMapping{{RestApiId: aws.String("api-1"), BasePath: aws.String("v1")}}},
		},
	}
	listCalls := []string{
		"GetDomainNames",
		"GetBasePathMappings a.example.com id=",
		"GetDomainNames",
		"GetBasePathMappings b.example.com id=abc123",
		"GetBasePathMappings b.example.com id=abc123",
	}

	tests := map[string]struct {
		mock          mockedApiGateway
		expectedCalls []string
		expectedErr   string
	}{
		"mappedApiLosesOnlyItsMappingsFirst": {
			mock: mockedApiGateway{DomainPages: domains, MappingPages: mappings},
			expectedCalls: append(append([]string{}, listCalls...),
				"DeleteBasePathMapping a.example.com id= (none)",
				"DeleteBasePathMapping b.example.com id=abc123 v1",
				"DeleteRestApi api-1",
			),
		},
		"noDomains": {
			mock:          mockedApiGateway{},
			expectedCalls: []string{"GetDomainNames", "DeleteRestApi api-1"},
		},
		"mappingDeleteFailsSoApiIsKept": {
			mock:          mockedApiGateway{DomainPages: domains, MappingPages: mappings, DeleteBasePathMappingErr: errors.New("access denied")},
			expectedCalls: listCalls,
			expectedErr:   "access denied",
		},
		"domainListFailsSoApiIsStillDeleted": {
			mock:          mockedApiGateway{GetDomainNamesErr: errors.New("access denied")},
			expectedCalls: []string{"GetDomainNames", "DeleteRestApi api-1"},
		},
		"oneDomainFailsToListSoTheOthersMappingsStillGo": {
			mock: mockedApiGateway{
				DomainPages:            domains,
				MappingPages:           mappings,
				GetBasePathMappingsErr: map[string]error{"a.example.com": errors.New("access denied")},
			},
			expectedCalls: append(append([]string{}, listCalls...),
				"DeleteBasePathMapping b.example.com id=abc123 v1",
				"DeleteRestApi api-1",
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			calls := &apiGatewayCallLog{}
			tc.mock.Calls = calls

			results, err := nukeApiGateway(tc.mock, "api-1")

			require.Len(t, results, 1)
			if tc.expectedErr == "" {
				require.NoError(t, err)
				require.NoError(t, results[0].Error)
			} else {
				require.ErrorContains(t, err, tc.expectedErr)
				require.ErrorContains(t, results[0].Error, tc.expectedErr)
			}
			require.Equal(t, tc.expectedCalls, calls.list())
		})
	}
}

func TestAPIGateway_NukeAll_ListsMappingsOncePerBatch(t *testing.T) {
	t.Parallel()

	calls := &apiGatewayCallLog{}
	mock := mockedApiGateway{
		DomainPages: []apigateway.GetDomainNamesOutput{
			{Items: []types.DomainName{{DomainName: aws.String("a.example.com")}}},
		},
		MappingPages: map[string][]apigateway.GetBasePathMappingsOutput{
			"a.example.com": {{Items: []types.BasePathMapping{
				{RestApiId: aws.String("api-1"), BasePath: aws.String("v1")},
				{RestApiId: aws.String("api-2"), BasePath: aws.String("v2")},
			}}},
		},
		Calls: calls,
	}

	results, err := nukeApiGateway(mock, "api-1", "api-2", "api-3")

	require.NoError(t, err)
	require.Len(t, results, 3)
	require.ElementsMatch(t, []string{
		"GetDomainNames",
		"GetBasePathMappings a.example.com id=",
		"DeleteBasePathMapping a.example.com id= v1",
		"DeleteRestApi api-1",
		"DeleteBasePathMapping a.example.com id= v2",
		"DeleteRestApi api-2",
		"DeleteRestApi api-3",
	}, calls.list())
}

// nukeApiGateway deletes the REST APIs through the resource's own nuker, as a run of cloud-nuke does.
func nukeApiGateway(client ApiGatewayAPI, ids ...string) ([]resource.NukeResult, error) {
	r := NewApiGateway().(*AwsResourceAdapter[ApiGatewayAPI]).Resource
	r.Client = client
	return r.Nuke(context.Background(), ids)
}
