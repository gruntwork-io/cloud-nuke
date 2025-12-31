package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// SecretsManagerAPI defines the interface for Secrets Manager operations.
type SecretsManagerAPI interface {
	ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
	RemoveRegionsFromReplication(ctx context.Context, params *secretsmanager.RemoveRegionsFromReplicationInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.RemoveRegionsFromReplicationOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

// NewSecretsManagerSecrets creates a new SecretsManagerSecrets resource using the generic resource pattern.
func NewSecretsManagerSecrets() AwsResource {
	return NewAwsResource(&resource.Resource[SecretsManagerAPI]{
		ResourceTypeName: "secretsmanager",
		// Tentative batch size to ensure AWS doesn't throttle. Note that secrets manager does not support bulk delete,
		// so we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
		// overloading the runtime and to avoid AWS throttling with many API calls.
		BatchSize: 10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SecretsManagerAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = secretsmanager.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SecretsManagerSecrets
		},
		Lister: listSecretsManagerSecrets,
		Nuker:  resource.SimpleBatchDeleter(deleteSecretsManagerSecret),
	})
}

// listSecretsManagerSecrets retrieves all Secrets Manager secrets that match the config filters.
func listSecretsManagerSecrets(ctx context.Context, client SecretsManagerAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allSecrets []*string
	paginator := secretsmanager.NewListSecretsPaginator(client, &secretsmanager.ListSecretsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, secret := range page.SecretList {
			if shouldIncludeSecret(&secret, cfg) {
				allSecrets = append(allSecrets, secret.ARN)
			}
		}
	}

	return allSecrets, nil
}

// shouldIncludeSecret determines if a secret should be included based on config filters.
func shouldIncludeSecret(secret *types.SecretListEntry, cfg config.ResourceType) bool {
	if secret == nil {
		return false
	}

	// reference time for excludeAfter is last accessed time, unless it was never accessed in which created time is
	// used.
	var referenceTime time.Time
	if secret.LastAccessedDate == nil {
		referenceTime = *secret.CreatedDate
	} else {
		referenceTime = *secret.LastAccessedDate
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Time: &referenceTime,
		Name: secret.Name,
		Tags: util.ConvertSecretsManagerTagsToMap(secret.Tags),
	})
}

// deleteSecretsManagerSecret deletes a single Secrets Manager secret.
// If this region's secret is primary and has replicated secrets, removes replication first.
func deleteSecretsManagerSecret(ctx context.Context, client SecretsManagerAPI, secretID *string) error {
	// Get secret details to check for replications
	secret, err := client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: secretID,
	})
	if err != nil {
		return err
	}

	// Delete replications if this is a primary secret with replicas
	if len(secret.ReplicationStatus) > 0 {
		replicationRegions := make([]string, 0, len(secret.ReplicationStatus))
		for _, replicationStatus := range secret.ReplicationStatus {
			replicationRegions = append(replicationRegions, *replicationStatus.Region)
		}

		_, err = client.RemoveRegionsFromReplication(ctx, &secretsmanager.RemoveRegionsFromReplicationInput{
			SecretId:             secretID,
			RemoveReplicaRegions: replicationRegions,
		})
		if err != nil {
			return err
		}
	}

	// Delete the secret
	_, err = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		ForceDeleteWithoutRecovery: aws.Bool(true),
		SecretId:                   secretID,
	})
	return err
}
