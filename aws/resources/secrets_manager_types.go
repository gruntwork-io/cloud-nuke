package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SecretsManagerSecretsAPI interface {
	ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
	RemoveRegionsFromReplication(ctx context.Context, params *secretsmanager.RemoveRegionsFromReplicationInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.RemoveRegionsFromReplicationOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

// SecretsManagerSecrets - represents all AWS secrets manager secrets that should be deleted.
type SecretsManagerSecrets struct {
	BaseAwsResource
	Client    SecretsManagerSecretsAPI
	Region    string
	SecretIDs []string
}

func (sms *SecretsManagerSecrets) Init(cfg aws.Config) {
	sms.Client = secretsmanager.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (sms *SecretsManagerSecrets) ResourceName() string {
	return "secretsmanager"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (sms *SecretsManagerSecrets) ResourceIdentifiers() []string {
	return sms.SecretIDs
}

func (sms *SecretsManagerSecrets) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SecretsManagerSecrets
}

func (sms *SecretsManagerSecrets) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that secrets manager does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (sms *SecretsManagerSecrets) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sms.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sms.SecretIDs = aws.ToStringSlice(identifiers)
	return sms.SecretIDs, nil
}

// Nuke - nuke 'em all!!!
func (sms *SecretsManagerSecrets) Nuke(identifiers []string) error {
	if err := sms.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
