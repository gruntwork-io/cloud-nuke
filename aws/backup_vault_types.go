package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup/backupiface"
	"github.com/gruntwork-io/go-commons/errors"
)

type BackupVault struct {
	Client backupiface.BackupAPI
	Region string
	Names  []string
}

// ResourceName - the simple name of the aws resource
func (bv BackupVault) ResourceName() string {
	return "backup-vault"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (bv BackupVault) ResourceIdentifiers() []string {
	return bv.Names
}

func (bv BackupVault) MaxBatchSize() int {
	return 50
}

// Nuke - nuke 'em all!!!
func (bv BackupVault) Nuke(session *session.Session, identifiers []string) error {
	if err := bv.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
