package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// BackupVaultAPI defines the interface for Backup Vault operations.
type BackupVaultAPI interface {
	DeleteBackupVault(ctx context.Context, params *backup.DeleteBackupVaultInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupVaultOutput, error)
	DeleteRecoveryPoint(ctx context.Context, params *backup.DeleteRecoveryPointInput, optFns ...func(*backup.Options)) (*backup.DeleteRecoveryPointOutput, error)
	ListBackupVaults(ctx context.Context, params *backup.ListBackupVaultsInput, optFns ...func(*backup.Options)) (*backup.ListBackupVaultsOutput, error)
	ListRecoveryPointsByBackupVault(ctx context.Context, params *backup.ListRecoveryPointsByBackupVaultInput, optFns ...func(*backup.Options)) (*backup.ListRecoveryPointsByBackupVaultOutput, error)
}

// NewBackupVault creates a new BackupVault resource using the generic resource pattern.
func NewBackupVault() AwsResource {
	return NewAwsResource(&resource.Resource[BackupVaultAPI]{
		ResourceTypeName: "backup-vault",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[BackupVaultAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = backup.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.BackupVault
		},
		Lister: listBackupVaults,
		Nuker:  resource.MultiStepDeleter(nukeRecoveryPoints, nukeBackupVault),
	})
}

// listBackupVaults retrieves all Backup Vaults that match the config filters.
func listBackupVaults(ctx context.Context, client BackupVaultAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string
	paginator := backup.NewListBackupVaultsPaginator(client, &backup.ListBackupVaultsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, backupVault := range page.BackupVaultList {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: backupVault.BackupVaultName,
				Time: backupVault.CreationDate,
			}) {
				names = append(names, backupVault.BackupVaultName)
			}
		}
	}

	return names, nil
}

// nukeRecoveryPoints deletes all recovery points in a backup vault.
func nukeRecoveryPoints(ctx context.Context, client BackupVaultAPI, name *string) error {
	logging.Debugf("Nuking the recovery points of backup vault %s", aws.ToString(name))

	output, err := client.ListRecoveryPointsByBackupVault(ctx, &backup.ListRecoveryPointsByBackupVaultInput{
		BackupVaultName: name,
	})
	if err != nil {
		logging.Debugf("[Failed] listing the recovery points of backup vault %s: %v", aws.ToString(name), err)
		return err
	}

	for _, recoveryPoint := range output.RecoveryPoints {
		logging.Debugf("Deleting recovery point %s from backup vault %s", aws.ToString(recoveryPoint.RecoveryPointArn), aws.ToString(name))
		_, err = client.DeleteRecoveryPoint(ctx, &backup.DeleteRecoveryPointInput{
			BackupVaultName:  name,
			RecoveryPointArn: recoveryPoint.RecoveryPointArn,
		})

		if err != nil {
			logging.Debugf("[Failed] nuking the backup vault %s: %v", aws.ToString(name), err)
			return err
		}
	}

	// wait until all the recovery points nuked successfully
	err = waitUntilRecoveryPointsDeleted(ctx, client, name)
	if err != nil {
		logging.Debugf("[Failed] waiting deletion of recovery points for backup vault %s: %v", aws.ToString(name), err)
		return err
	}

	logging.Debugf("[Ok] successfully nuked recovery points of backup vault %s", aws.ToString(name))

	return nil
}

// waitUntilRecoveryPointsDeleted waits until all recovery points are deleted.
func waitUntilRecoveryPointsDeleted(ctx context.Context, client BackupVaultAPI, name *string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("recovery point deletion check timed out after 1 minute")
		case <-ticker.C:
			output, err := client.ListRecoveryPointsByBackupVault(ctx, &backup.ListRecoveryPointsByBackupVaultInput{
				BackupVaultName: name,
			})
			if err != nil {
				logging.Debugf("recovery point(s) existance checking error : %v", err)
				return err
			}

			if len(output.RecoveryPoints) == 0 {
				return nil
			}
			logging.Debugf("%v Recovery point(s) still exists, waiting...", len(output.RecoveryPoints))
		}
	}
}

// nukeBackupVault deletes a single backup vault.
func nukeBackupVault(ctx context.Context, client BackupVaultAPI, name *string) error {
	_, err := client.DeleteBackupVault(ctx, &backup.DeleteBackupVaultInput{
		BackupVaultName: name,
	})
	if err != nil {
		logging.Debugf("[Failed] nuking the backup vault %s: %v", aws.ToString(name), err)
		return err
	}
	return nil
}
