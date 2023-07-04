package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

type BackupVault struct {
	Names []string
}

// ResourceName - the simple name of the aws resource
func (ct BackupVault) ResourceName() string {
	return "backup-vault"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ct BackupVault) ResourceIdentifiers() []string {
	return ct.Names
}

func (ct BackupVault) MaxBatchSize() int {
	return 50
}

// Nuke - nuke 'em all!!!
func (ct BackupVault) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllBackupVaults(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
