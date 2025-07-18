package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedOIDCProvider struct {
	OIDCProvidersAPI
	ListOpenIDConnectProvidersOutput  iam.ListOpenIDConnectProvidersOutput
	GetOpenIDConnectProviderOutput    map[string]iam.GetOpenIDConnectProviderOutput
	DeleteOpenIDConnectProviderOutput iam.DeleteOpenIDConnectProviderOutput
}

func (m mockedOIDCProvider) DeleteOpenIDConnectProvider(ctx context.Context, params *iam.DeleteOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.DeleteOpenIDConnectProviderOutput, error) {
	return &m.DeleteOpenIDConnectProviderOutput, nil
}

func (m mockedOIDCProvider) ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput, optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
	return &m.ListOpenIDConnectProvidersOutput, nil
}

func (m mockedOIDCProvider) GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
	arn := params.OpenIDConnectProviderArn
	resp := m.GetOpenIDConnectProviderOutput[*arn]

	return &resp, nil
}

func TestOIDCProvider_GetAll(t *testing.T) {

	t.Parallel()

	testArn1 := "test-arn1"
	testArn2 := "test-arn2"
	testUrl1 := "https://test1.com"
	testUrl2 := "https://test2.com"
	now := time.Now()
	oidcp := OIDCProviders{
		Client: mockedOIDCProvider{
			ListOpenIDConnectProvidersOutput: iam.ListOpenIDConnectProvidersOutput{
				OpenIDConnectProviderList: []types.OpenIDConnectProviderListEntry{
					{Arn: aws.String(testArn1)},
					{Arn: aws.String(testArn2)},
				},
			},
			GetOpenIDConnectProviderOutput: map[string]iam.GetOpenIDConnectProviderOutput{
				testArn1: {
					Url:        aws.String(testUrl1),
					CreateDate: aws.Time(now),
					Tags:       []types.Tag{{Key: aws.String("foo"), Value: aws.String("bar")}},
				},
				testArn2: {
					Url:        aws.String(testUrl2),
					CreateDate: aws.Time(now.Add(1)),
					Tags:       []types.Tag{{Key: aws.String("faz"), Value: aws.String("baz")}},
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
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testUrl1),
					}}},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testArn1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": config.Expression{RE: *regexp.MustCompile("bar")}},
				}},
			expected: []string{testArn2},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": config.Expression{RE: *regexp.MustCompile("bar")}},
				}},
			expected: []string{testArn1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := oidcp.getAll(context.Background(), config.Config{
				OIDCProvider: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestOIDCProvider_NukeAll(t *testing.T) {

	t.Parallel()

	oidcp := OIDCProviders{
		Client: mockedOIDCProvider{
			DeleteOpenIDConnectProviderOutput: iam.DeleteOpenIDConnectProviderOutput{},
		},
	}

	err := oidcp.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
