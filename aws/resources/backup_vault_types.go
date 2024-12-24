package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type BackupVaultAPI interface {
	DeleteBackupVault(ctx context.Context, params *backup.DeleteBackupVaultInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupVaultOutput, error)
	DeleteRecoveryPoint(ctx context.Context, params *backup.DeleteRecoveryPointInput, optFns ...func(*backup.Options)) (*backup.DeleteRecoveryPointOutput, error)
	ListBackupVaults(ctx context.Context, params *backup.ListBackupVaultsInput, optFns ...func(*backup.Options)) (*backup.ListBackupVaultsOutput, error)
	ListRecoveryPointsByBackupVault(ctx context.Context, params *backup.ListRecoveryPointsByBackupVaultInput, optFns ...func(*backup.Options)) (*backup.ListRecoveryPointsByBackupVaultOutput, error)
}

type BackupVault struct {
	BaseAwsResource
	Client BackupVaultAPI
	Region string
	Names  []string
}

func (bv *BackupVault) Init(cfg aws.Config) {
	bv.Client = backup.NewFromConfig(cfg)
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

func (bv *BackupVault) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.BackupVault
}
func (bv *BackupVault) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := bv.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	bv.Names = aws.ToStringSlice(identifiers)
	return bv.Names, nil
}

// Nuke - nuke 'em all!!!
func (bv *BackupVault) Nuke(identifiers []string) error {
	if err := bv.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
