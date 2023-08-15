package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// OIDCProviders - represents all AWS OpenID Connect providers that should be deleted.
type OIDCProviders struct {
	Client       iamiface.IAMAPI
	ProviderARNs []string
}

func (oidcprovider *OIDCProviders) Init(session *session.Session) {
	oidcprovider.Client = iam.New(session)
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

func (oidcprovider *OIDCProviders) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := oidcprovider.getAll(configObj)
	if err != nil {
		return nil, err
	}

	oidcprovider.ProviderARNs = awsgo.StringValueSlice(identifiers)
	return oidcprovider.ProviderARNs, nil
}

// Nuke - nuke 'em all!!!
func (oidcprovider *OIDCProviders) Nuke(identifiers []string) error {
	if err := oidcprovider.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
