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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedOIDCProvider struct {
	OIDCProvidersAPI
	ListOutput  iam.ListOpenIDConnectProvidersOutput
	GetOutputs  map[string]iam.GetOpenIDConnectProviderOutput
	DeleteError error
}

func (m mockedOIDCProvider) ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput, optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error) {
	return &m.ListOutput, nil
}

func (m mockedOIDCProvider) GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
	resp := m.GetOutputs[aws.ToString(params.OpenIDConnectProviderArn)]
	return &resp, nil
}

func (m mockedOIDCProvider) DeleteOpenIDConnectProvider(ctx context.Context, params *iam.DeleteOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.DeleteOpenIDConnectProviderOutput, error) {
	return &iam.DeleteOpenIDConnectProviderOutput{}, m.DeleteError
}

func TestOIDCProvider_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testArn1 := "arn:aws:iam::123456789012:oidc-provider/test1.example.com"
	testArn2 := "arn:aws:iam::123456789012:oidc-provider/test2.example.com"
	testUrl1 := "https://test1.example.com"
	testUrl2 := "https://test2.example.com"

	client := mockedOIDCProvider{
		ListOutput: iam.ListOpenIDConnectProvidersOutput{
			OpenIDConnectProviderList: []types.OpenIDConnectProviderListEntry{
				{Arn: aws.String(testArn1)},
				{Arn: aws.String(testArn2)},
			},
		},
		GetOutputs: map[string]iam.GetOpenIDConnectProviderOutput{
			testArn1: {
				Url:        aws.String(testUrl1),
				CreateDate: aws.Time(now),
				Tags:       []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
			},
			testArn2: {
				Url:        aws.String(testUrl2),
				CreateDate: aws.Time(now.Add(time.Hour)),
				Tags:       []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"no filter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"name exclusion": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test1")}},
				},
			},
			expected: []string{testArn2},
		},
		"time filter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testArn1},
		},
		"tag filter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testArn1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := listOIDCProviders(context.Background(), client, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(result))
		})
	}
}

func TestOIDCProvider_Delete(t *testing.T) {
	t.Parallel()

	client := mockedOIDCProvider{}
	err := deleteOIDCProvider(context.Background(), client, aws.String("test-arn"))
	require.NoError(t, err)
}
