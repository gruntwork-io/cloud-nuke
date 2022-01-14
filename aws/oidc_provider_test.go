package aws

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: OpenID Connect Provider is a global resource, so we use the default region for the tests.

func TestListOIDCProviders(t *testing.T) {
	t.Parallel()

	session, err := session.NewSession(&aws.Config{Region: aws.String(defaultRegion)})
	require.NoError(t, err)
	svc := iam.New(session)

	oidcProviderARN := createOIDCProvider(t, svc, "base", defaultRegion)
	defer deleteOIDCProvider(t, svc, oidcProviderARN, true)

	providerARNs, err := getAllOIDCProviders(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(providerARNs), aws.StringValue(oidcProviderARN))
}

func TestTimeFilterExclusionNewlyCreatedOIDCProvider(t *testing.T) {
	t.Parallel()

	session, err := session.NewSession(&aws.Config{Region: aws.String(defaultRegion)})
	require.NoError(t, err)
	svc := iam.New(session)

	oidcProviderARN := createOIDCProvider(t, svc, "base", defaultRegion)
	defer deleteOIDCProvider(t, svc, oidcProviderARN, true)

	// Assert OpenID Connect Provider is picked up without filters
	providerARNsNewer, err := getAllOIDCProviders(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(providerARNsNewer), aws.StringValue(oidcProviderARN))

	// Assert provider doesn't appear when we look at providers older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	providerARNsOlder, err := getAllOIDCProviders(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(providerARNsOlder), aws.StringValue(oidcProviderARN))
}

func TestConfigExclusionCreatedOIDCProvider(t *testing.T) {
	t.Parallel()

	session, err := session.NewSession(&aws.Config{Region: aws.String(defaultRegion)})
	require.NoError(t, err)
	svc := iam.New(session)

	includedOIDCProviderARN := createOIDCProvider(t, svc, "include", defaultRegion)
	defer deleteOIDCProvider(t, svc, includedOIDCProviderARN, true)

	excludedOIDCProviderARN := createOIDCProvider(t, svc, "exclude", defaultRegion)
	defer deleteOIDCProvider(t, svc, excludedOIDCProviderARN, true)

	// Assert OpenID Connect Providers are picked up without filters
	providerARNsNewer, err := getAllOIDCProviders(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(providerARNsNewer), aws.StringValue(includedOIDCProviderARN))
	assert.Contains(t, aws.StringValueSlice(providerARNsNewer), aws.StringValue(excludedOIDCProviderARN))

	// Assert provider doesn't appear when we filter providers by config file
	providerARNsConfigFiltered, err := getAllOIDCProviders(session, time.Now(), config.Config{
		OIDCProvider: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					config.Expression{
						RE: *regexp.MustCompile(".*include.*"),
					},
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(providerARNsConfigFiltered), aws.StringValue(includedOIDCProviderARN))
	assert.NotContains(t, aws.StringValueSlice(providerARNsConfigFiltered), aws.StringValue(excludedOIDCProviderARN))
}

func TestNukeOIDCProviderOne(t *testing.T) {
	t.Parallel()

	session, err := session.NewSession(&aws.Config{Region: aws.String(defaultRegion)})
	require.NoError(t, err)
	svc := iam.New(session)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	oidcProviderARN := createOIDCProvider(t, svc, "base", defaultRegion)
	defer deleteOIDCProvider(t, svc, oidcProviderARN, false)

	identifiers := []*string{oidcProviderARN}
	require.NoError(
		t,
		nukeAllOIDCProviders(session, identifiers),
	)

	// Make sure the OIDC Provider is deleted.
	assertOIDCProvidersDeleted(t, svc, identifiers)
}

func TestNukeOIDCProviderMoreThanOne(t *testing.T) {
	t.Parallel()

	session, err := session.NewSession(&aws.Config{Region: aws.String(defaultRegion)})
	require.NoError(t, err)
	svc := iam.New(session)

	providers := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		oidcProviderARN := createOIDCProvider(t, svc, "base", defaultRegion)
		defer deleteOIDCProvider(t, svc, oidcProviderARN, false)
		providers = append(providers, oidcProviderARN)
	}

	require.NoError(
		t,
		nukeAllOIDCProviders(session, providers),
	)

	// Make sure all OIDCProviders are deleted.
	assertOIDCProvidersDeleted(t, svc, providers)
}

// Helper functions for driving the OIDC Provider tests

// createOIDCProvider will create a new OIDC Provider
func createOIDCProvider(t *testing.T, svc *iam.IAM, basename string, region string) *string {
	input := &iam.CreateOpenIDConnectProviderInput{
		Url: aws.String(fmt.Sprintf("https://%s.%s.gruntwork-sandbox.in", random.UniqueId(), basename)),
		// We can use a non-functional thumbprint here because we don't care if the provider actually works - only that
		// the resource exists.
		ThumbprintList: aws.StringSlice([]string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}),
	}
	resp, err := svc.CreateOpenIDConnectProvider(input)
	require.NoError(t, err)

	// Wait 10 seconds after creation to ensure the OIDC provider gets propagated through AWS system.
	time.Sleep(time.Second * 10)

	return resp.OpenIDConnectProviderArn
}

// deleteOIDCProvider is a function to delete the given OpenID Connect Provider.
func deleteOIDCProvider(t *testing.T, svc *iam.IAM, providerARN *string, checkErr bool) {
	input := &iam.DeleteOpenIDConnectProviderInput{OpenIDConnectProviderArn: providerARN}
	_, err := svc.DeleteOpenIDConnectProvider(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func assertOIDCProvidersDeleted(t *testing.T, svc *iam.IAM, identifiers []*string) {
	resp, err := svc.ListOpenIDConnectProviders(&iam.ListOpenIDConnectProvidersInput{})
	require.NoError(t, err)

	providerARNsFound := []string{}
	for _, provider := range resp.OpenIDConnectProviderList {
		if provider != nil {
			providerARNsFound = append(providerARNsFound, aws.StringValue(provider.Arn))
		}
	}

	for _, providerARN := range identifiers {
		assert.NotContains(t, providerARNsFound, aws.StringValue(providerARN))
	}
}
