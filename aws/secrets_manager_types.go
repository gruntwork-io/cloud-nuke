package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// SecretsManagerSecrets - represents all AWS secrets manager secrets that should be deleted.
type SecretsManagerSecrets struct {
	SecretIDs []string
}

// ResourceName - the simple name of the aws resource
func (secret SecretsManagerSecrets) ResourceName() string {
	return "secretsmanager"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (secret SecretsManagerSecrets) ResourceIdentifiers() []string {
	return secret.SecretIDs
}

func (secret SecretsManagerSecrets) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that secrets manager does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

// Nuke - nuke 'em all!!!
func (secret SecretsManagerSecrets) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSecretsManagerSecrets(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
