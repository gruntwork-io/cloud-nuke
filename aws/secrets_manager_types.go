package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/gruntwork-io/go-commons/errors"
)

// SecretsManagerSecret - represents all AWS secrets manager secrets that should be deleted.
type SecretsManagerSecret struct {
	Client    secretsmanageriface.SecretsManagerAPI
	Region    string
	SecretIDs []string
}

// ResourceName - the simple name of the aws resource
func (secret SecretsManagerSecret) ResourceName() string {
	return "secrets-manager"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (secret SecretsManagerSecret) ResourceIdentifiers() []string {
	return secret.SecretIDs
}

func (secret SecretsManagerSecret) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that secrets manager does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

// Nuke - nuke 'em all!!!
func (secret SecretsManagerSecret) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSecretsManagerSecrets(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
