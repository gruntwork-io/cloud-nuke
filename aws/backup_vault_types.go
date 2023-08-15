package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/aws/aws-sdk-go/service/backup/backupiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type BackupVault struct {
	Client backupiface.BackupAPI
	Region string
	Names  []string
}

func (bv *BackupVault) Init(session *session.Session) {
	bv.Client = backup.New(session)
}

// ResourceName - the simple name of the aws resource
func (bv *BackupVault) ResourceName() string {
	return "backup-vault"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (bv *BackupVault) ResourceIdentifiers() []string {
	return bv.Names
}

func (bv *BackupVault) MaxBatchSize() int {
	return 50
}

func (bv *BackupVault) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := bv.getAll(configObj)
	if err != nil {
		return nil, err
	}

	bv.Names = awsgo.StringValueSlice(identifiers)
	return bv.Names, nil
}

// Nuke - nuke 'em all!!!
func (bv *BackupVault) Nuke(identifiers []string) error {
	if err := bv.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
