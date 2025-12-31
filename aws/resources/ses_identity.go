package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// SESIdentityAPI defines the interface for SES Identity operations.
type SESIdentityAPI interface {
	ListIdentities(ctx context.Context, params *ses.ListIdentitiesInput, optFns ...func(*ses.Options)) (*ses.ListIdentitiesOutput, error)
	DeleteIdentity(ctx context.Context, params *ses.DeleteIdentityInput, optFns ...func(*ses.Options)) (*ses.DeleteIdentityOutput, error)
}

// NewSesIdentities creates a new SesIdentities resource using the generic resource pattern.
func NewSesIdentities() AwsResource {
	return NewAwsResource(&resource.Resource[SESIdentityAPI]{
		ResourceTypeName: "ses-identity",
		BatchSize:        maxBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SESIdentityAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ses.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SESIdentity
		},
		Lister: listSesIdentities,
		Nuker:  resource.SimpleBatchDeleter(deleteSesIdentity),
	})
}

// listSesIdentities retrieves all SES Identities that match the config filters.
func listSesIdentities(ctx context.Context, client SESIdentityAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.ListIdentities(ctx, &ses.ListIdentitiesInput{})
	if err != nil {
		return nil, err
	}

	var ids []*string
	for _, id := range result.Identities {
		if cfg.ShouldInclude(config.ResourceValue{Name: aws.String(id)}) {
			ids = append(ids, aws.String(id))
		}
	}

	return ids, nil
}

// deleteSesIdentity deletes a single SES Identity.
func deleteSesIdentity(ctx context.Context, client SESIdentityAPI, id *string) error {
	_, err := client.DeleteIdentity(ctx, &ses.DeleteIdentityInput{
		Identity: id,
	})
	return err
}
