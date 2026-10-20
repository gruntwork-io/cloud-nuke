package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedApiGatewayV2 struct {
	ApiGatewayV2API
	GetApisOutput   apigatewayv2.GetApisOutput
	DeleteApiOutput apigatewayv2.DeleteApiOutput
	// DomainPages are returned by page token.
	DomainPages       []apigatewayv2.GetDomainNamesOutput
	GetDomainNamesErr error
	// MappingPages are keyed by domain name and returned by page token.
	MappingPages map[string][]apigatewayv2.GetApiMappingsOutput
	// GetApiMappingsErr fails the listing on the domains it names.
	GetApiMappingsErr   map[string]error
	DeleteApiMappingErr error
	Calls               *apiGatewayCallLog
}

func (m mockedApiGatewayV2) GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error) {
	return &m.GetApisOutput, nil
}

func (m mockedApiGatewayV2) DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error) {
	m.Calls.add("DeleteApi " + aws.ToString(params.ApiId))
	return &m.DeleteApiOutput, nil
}

func (m mockedApiGatewayV2) GetDomainNames(ctx context.Context, params *apigatewayv2.GetDomainNamesInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetDomainNamesOutput, error) {
	m.Calls.add("GetDomainNames")
	if m.GetDomainNamesErr != nil {
		return nil, m.GetDomainNamesErr
	}
	if len(m.DomainPages) == 0 {
		return &apigatewayv2.GetDomainNamesOutput{}, nil
	}
	return &m.DomainPages[apiGatewayPage(params.NextToken)], nil
}

func (m mockedApiGatewayV2) GetApiMappings(ctx context.Context, params *apigatewayv2.GetApiMappingsInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApiMappingsOutput, error) {
	if err := m.GetApiMappingsErr[aws.ToString(params.DomainName)]; err != nil {
		return nil, err
	}
	pages := m.MappingPages[aws.ToString(params.DomainName)]
	if len(pages) == 0 {
		return &apigatewayv2.GetApiMappingsOutput{}, nil
	}
	return &pages[apiGatewayPage(params.NextToken)], nil
}

func (m mockedApiGatewayV2) DeleteApiMapping(ctx context.Context, params *apigatewayv2.DeleteApiMappingInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiMappingOutput, error) {
	if m.DeleteApiMappingErr != nil {
		return nil, m.DeleteApiMappingErr
	}
	m.Calls.add("DeleteApiMapping " + aws.ToString(params.DomainName) + " " + aws.ToString(params.ApiMappingId))
	return &apigatewayv2.DeleteApiMappingOutput{}, nil
}

func TestApiGatewayV2_GetAll(t *testing.T) {
	t.Parallel()

	testApiID1 := "api-1"
	testApiID2 := "api-2"
	testApiName1 := "test-api-1"
	testApiName2 := "test-api-2"
	now := time.Now()

	mock := mockedApiGatewayV2{
		GetApisOutput: apigatewayv2.GetApisOutput{
			Items: []types.Api{
				{
					ApiId:       aws.String(testApiID1),
					Name:        aws.String(testApiName1),
					CreatedDate: aws.Time(now),
					Tags:        map[string]string{"env": "dev"},
				},
				{
					ApiId:       aws.String(testApiID2),
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
			apis, err := listApiGatewaysV2(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(apis))
		})
	}
}

func TestApiGatewayV2_NukeAll(t *testing.T) {
	t.Parallel()

	// Two domains on two pages; api-1 is mapped on both, the second time on the second page of mappings.
	domains := []apigatewayv2.GetDomainNamesOutput{
		{Items: []types.DomainName{{DomainName: aws.String("a.example.com")}}, NextToken: aws.String("1")},
		{Items: []types.DomainName{{DomainName: aws.String("b.example.com")}}},
	}
	mappings := map[string][]apigatewayv2.GetApiMappingsOutput{
		"a.example.com": {
			{Items: []types.ApiMapping{
				{ApiId: aws.String("api-1"), ApiMappingId: aws.String("map-a1")},
				{ApiId: aws.String("api-2"), ApiMappingId: aws.String("map-a2")},
			}},
		},
		"b.example.com": {
			{Items: []types.ApiMapping{{ApiId: aws.String("api-2"), ApiMappingId: aws.String("map-b2")}}, NextToken: aws.String("1")},
			{Items: []types.ApiMapping{{ApiId: aws.String("api-1"), ApiMappingId: aws.String("map-b1")}}},
		},
	}

	tests := map[string]struct {
		mock          mockedApiGatewayV2
		expectedCalls []string
		expectedErr   string
	}{
		"mappedApiLosesOnlyItsMappingsFirst": {
			mock: mockedApiGatewayV2{DomainPages: domains, MappingPages: mappings},
			expectedCalls: []string{
				"GetDomainNames",
				"GetDomainNames",
				"DeleteApiMapping a.example.com map-a1",
				"DeleteApiMapping b.example.com map-b1",
				"DeleteApi api-1",
			},
		},
		"noDomains": {
			mock:          mockedApiGatewayV2{},
			expectedCalls: []string{"GetDomainNames", "DeleteApi api-1"},
		},
		"mappingDeleteFailsSoApiIsKept": {
			mock:          mockedApiGatewayV2{DomainPages: domains, MappingPages: mappings, DeleteApiMappingErr: errors.New("access denied")},
			expectedCalls: []string{"GetDomainNames", "GetDomainNames"},
			expectedErr:   "access denied",
		},
		"domainListFailsSoApiIsStillDeleted": {
			mock:          mockedApiGatewayV2{GetDomainNamesErr: errors.New("access denied")},
			expectedCalls: []string{"GetDomainNames", "DeleteApi api-1"},
		},
		"oneDomainFailsToListSoTheOthersMappingsStillGo": {
			mock: mockedApiGatewayV2{
				DomainPages:       domains,
				MappingPages:      mappings,
				GetApiMappingsErr: map[string]error{"a.example.com": errors.New("access denied")},
			},
			expectedCalls: []string{
				"GetDomainNames",
				"GetDomainNames",
				"DeleteApiMapping b.example.com map-b1",
				"DeleteApi api-1",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			calls := &apiGatewayCallLog{}
			tc.mock.Calls = calls

			results, err := nukeApiGatewayV2(tc.mock, "api-1")

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

func TestApiGatewayV2_NukeAll_ListsMappingsOncePerBatch(t *testing.T) {
	t.Parallel()

	calls := &apiGatewayCallLog{}
	mock := mockedApiGatewayV2{
		DomainPages: []apigatewayv2.GetDomainNamesOutput{
			{Items: []types.DomainName{{DomainName: aws.String("a.example.com")}}},
		},
		MappingPages: map[string][]apigatewayv2.GetApiMappingsOutput{
			"a.example.com": {{Items: []types.ApiMapping{
				{ApiId: aws.String("api-1"), ApiMappingId: aws.String("map-a1")},
				{ApiId: aws.String("api-2"), ApiMappingId: aws.String("map-a2")},
			}}},
		},
		Calls: calls,
	}

	results, err := nukeApiGatewayV2(mock, "api-1", "api-2", "api-3")

	require.NoError(t, err)
	require.Len(t, results, 3)
	require.ElementsMatch(t, []string{
		"GetDomainNames",
		"DeleteApiMapping a.example.com map-a1",
		"DeleteApi api-1",
		"DeleteApiMapping a.example.com map-a2",
		"DeleteApi api-2",
		"DeleteApi api-3",
	}, calls.list())
}

// nukeApiGatewayV2 deletes the APIs through the resource's own nuker, as a run of cloud-nuke does.
func nukeApiGatewayV2(client ApiGatewayV2API, ids ...string) ([]resource.NukeResult, error) {
	r := NewApiGatewayV2().(*AwsResourceAdapter[ApiGatewayV2API]).Resource
	r.Client = client
	return r.Nuke(context.Background(), ids)
}
