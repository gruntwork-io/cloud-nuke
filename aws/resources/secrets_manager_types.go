package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/gruntwork-io/go-commons/errors"
)

// SecretsManagerSecrets - represents all AWS secrets manager secrets that should be deleted.
type SecretsManagerSecrets struct {
	Client    secretsmanageriface.SecretsManagerAPI
	Region    string
	SecretIDs []string
}

func (sms *SecretsManagerSecrets) Init(session *session.Session) {
	sms.Client = secretsmanager.New(session)
}

// ResourceName - the simple name of the aws resource
func (sms *SecretsManagerSecrets) ResourceName() string {
	return "secretsmanager"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (sms *SecretsManagerSecrets) ResourceIdentifiers() []string {
	return sms.SecretIDs
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

	sms.SecretIDs = awsgo.StringValueSlice(identifiers)
	return sms.SecretIDs, nil
}

// Nuke - nuke 'em all!!!
func (sms *SecretsManagerSecrets) Nuke(identifiers []string) error {
	if err := sms.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
