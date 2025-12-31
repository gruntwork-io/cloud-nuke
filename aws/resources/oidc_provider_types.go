package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type OIDCProvidersAPI interface {
	ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput, optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error)
	GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	DeleteOpenIDConnectProvider(ctx context.Context, params *iam.DeleteOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.DeleteOpenIDConnectProviderOutput, error)
}

// OIDCProviders - represents all AWS OpenID Connect providers that should be deleted.
type OIDCProviders struct {
	BaseAwsResource
	Client       OIDCProvidersAPI
	ProviderARNs []string
}

func (oidcprovider *OIDCProviders) Init(cfg aws.Config) {
	oidcprovider.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (oidcprovider *OIDCProviders) ResourceName() string {
	return "oidcprovider"
}

// ResourceIdentifiers - The ARNs of the OIDC providers.
func (oidcprovider *OIDCProviders) ResourceIdentifiers() []string {
	return oidcprovider.ProviderARNs
}

func (oidcprovider *OIDCProviders) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that OIDC Provider does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (oidcprovider *OIDCProviders) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.OIDCProvider
}

func (oidcprovider *OIDCProviders) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := oidcprovider.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	oidcprovider.ProviderARNs = aws.ToStringSlice(identifiers)
	return oidcprovider.ProviderARNs, nil
}

// Nuke - nuke 'em all!!!
func (oidcprovider *OIDCProviders) Nuke(ctx context.Context, identifiers []string) error {
	if err := oidcprovider.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
