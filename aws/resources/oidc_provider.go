package resources

import (
	"context"
	goerr "errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// OIDCProvidersAPI defines the interface for OIDC provider operations.
type OIDCProvidersAPI interface {
	ListOpenIDConnectProviders(ctx context.Context, params *iam.ListOpenIDConnectProvidersInput, optFns ...func(*iam.Options)) (*iam.ListOpenIDConnectProvidersOutput, error)
	GetOpenIDConnectProvider(ctx context.Context, params *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	DeleteOpenIDConnectProvider(ctx context.Context, params *iam.DeleteOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.DeleteOpenIDConnectProviderOutput, error)
}

// NewOIDCProviders creates a new OIDC Providers resource using the generic resource pattern.
// OIDC Providers are global IAM resources.
func NewOIDCProviders() AwsResource {
	return NewAwsResource(&resource.Resource[OIDCProvidersAPI]{
		ResourceTypeName: "oidcprovider",
		BatchSize:        10,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[OIDCProvidersAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.OIDCProvider
		},
		Lister: listOIDCProviders,
		Nuker:  resource.SimpleBatchDeleter(deleteOIDCProvider),
	})
}

// oidcProviderDetail holds the information needed for filtering OIDC providers.
// The list API only returns ARNs, so we need to fetch details for each provider.
type oidcProviderDetail struct {
	ARN         *string
	CreateTime  *time.Time
	ProviderURL *string
	Tags        map[string]string
}

// listOIDCProviders retrieves all OIDC providers that match the config filters.
// Note: ListOpenIDConnectProviders does not support pagination as it returns all providers in one call.
// Details must be fetched individually using GetOpenIDConnectProvider.
func listOIDCProviders(ctx context.Context, client OIDCProvidersAPI, _ resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var providerARNs []*string
	for _, provider := range output.OpenIDConnectProviderList {
		providerARNs = append(providerARNs, provider.Arn)
	}

	// Fetch details concurrently for filtering
	providers, err := getOIDCProviderDetails(ctx, client, providerARNs)
	if err != nil {
		return nil, err
	}

	var result []*string
	for _, provider := range providers {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: provider.ProviderURL,
			Time: provider.CreateTime,
			Tags: provider.Tags,
		}) {
			result = append(result, provider.ARN)
		}
	}

	return result, nil
}

// getOIDCProviderDetails fetches details for all providers concurrently.
func getOIDCProviderDetails(ctx context.Context, client OIDCProvidersAPI, arns []*string) ([]oidcProviderDetail, error) {
	if len(arns) == 0 {
		return nil, nil
	}

	type result struct {
		detail *oidcProviderDetail
		err    error
	}

	results := make([]result, len(arns))
	var wg sync.WaitGroup

	for i, arn := range arns {
		wg.Add(1)
		go func(idx int, providerARN *string) {
			defer wg.Done()

			resp, err := client.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{
				OpenIDConnectProviderArn: providerARN,
			})
			if err != nil {
				// If provider was deleted between list and get, ignore it
				var notFound *types.NoSuchEntityException
				if goerr.As(err, &notFound) {
					return
				}
				results[idx] = result{err: errors.WithStackTrace(err)}
				return
			}

			results[idx] = result{
				detail: &oidcProviderDetail{
					ARN:         providerARN,
					CreateTime:  resp.CreateDate,
					ProviderURL: resp.Url,
					Tags:        util.ConvertIAMTagsToMap(resp.Tags),
				},
			}
		}(i, arn)
	}

	wg.Wait()

	// Collect errors and results
	var allErrs *multierror.Error
	var details []oidcProviderDetail

	for _, r := range results {
		if r.err != nil {
			allErrs = multierror.Append(allErrs, r.err)
		} else if r.detail != nil {
			details = append(details, *r.detail)
		}
	}

	if err := allErrs.ErrorOrNil(); err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return details, nil
}

// deleteOIDCProvider deletes a single OIDC provider.
func deleteOIDCProvider(ctx context.Context, client OIDCProvidersAPI, arn *string) error {
	_, err := client.DeleteOpenIDConnectProvider(ctx, &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: arn,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
