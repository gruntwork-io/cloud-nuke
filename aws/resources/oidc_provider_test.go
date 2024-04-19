package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedOIDCProvider struct {
	iamiface.IAMAPI
	ListOpenIDConnectProvidersOutput  iam.ListOpenIDConnectProvidersOutput
	GetOpenIDConnectProviderOutput    map[string]iam.GetOpenIDConnectProviderOutput
	DeleteOpenIDConnectProviderOutput iam.DeleteOpenIDConnectProviderOutput
}

func (m mockedOIDCProvider) DeleteOpenIDConnectProvider(input *iam.DeleteOpenIDConnectProviderInput) (*iam.DeleteOpenIDConnectProviderOutput, error) {
	return &m.DeleteOpenIDConnectProviderOutput, nil
}

func (m mockedOIDCProvider) ListOpenIDConnectProviders(input *iam.ListOpenIDConnectProvidersInput) (*iam.ListOpenIDConnectProvidersOutput, error) {
	return &m.ListOpenIDConnectProvidersOutput, nil
}

func (m mockedOIDCProvider) GetOpenIDConnectProvider(input *iam.GetOpenIDConnectProviderInput) (*iam.GetOpenIDConnectProviderOutput, error) {
	arn := input.OpenIDConnectProviderArn
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
				OpenIDConnectProviderList: []*iam.OpenIDConnectProviderListEntry{
					{Arn: aws.String(testArn1)},
					{Arn: aws.String(testArn2)},
				},
			},
			GetOpenIDConnectProviderOutput: map[string]iam.GetOpenIDConnectProviderOutput{
				testArn1: {
					Url:        aws.String(testUrl1),
					CreateDate: aws.Time(now),
				},
				testArn2: {
					Url:        aws.String(testUrl2),
					CreateDate: aws.Time(now.Add(1)),
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := oidcp.getAll(context.Background(), config.Config{
				OIDCProvider: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
