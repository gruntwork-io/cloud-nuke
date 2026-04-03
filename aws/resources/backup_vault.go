package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// BackupVaultAPI defines the interface for Backup Vault operations.
type BackupVaultAPI interface {
	DeleteBackupVault(ctx context.Context, params *backup.DeleteBackupVaultInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupVaultOutput, error)
	DeleteRecoveryPoint(ctx context.Context, params *backup.DeleteRecoveryPointInput, optFns ...func(*backup.Options)) (*backup.DeleteRecoveryPointOutput, error)
	ListBackupVaults(ctx context.Context, params *backup.ListBackupVaultsInput, optFns ...func(*backup.Options)) (*backup.ListBackupVaultsOutput, error)
	ListRecoveryPointsByBackupVault(ctx context.Context, params *backup.ListRecoveryPointsByBackupVaultInput, optFns ...func(*backup.Options)) (*backup.ListRecoveryPointsByBackupVaultOutput, error)
	ListTags(ctx context.Context, params *backup.ListTagsInput, optFns ...func(*backup.Options)) (*backup.ListTagsOutput, error)
}

// NewBackupVault creates a new BackupVault resource using the generic resource pattern.
func NewBackupVault() AwsResource {
	return NewAwsResource(&resource.Resource[BackupVaultAPI]{
		ResourceTypeName: "backup-vault",
		BatchSize:        DefaultBatchSize,
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
			name := aws.ToString(backupVault.BackupVaultName)
			if strings.EqualFold(name, "default") || strings.HasPrefix(name, "aws/") {
				logging.Debugf("Skipping %s since it is a default backup vault", name)
				continue
			}

			tags, err := getTags(ctx, client, cfg, backupVault)
			if err != nil {
				logging.Errorf("Unable to fetch tags for %s: %s", aws.ToString(backupVault.BackupVaultArn), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: backupVault.BackupVaultName,
				Time: backupVault.CreationDate,
				Tags: tags,
			}) {
				names = append(names, backupVault.BackupVaultName)
			}
		}
	}

	return names, nil
}

// getTags retrieves the tags for a given backup vault if tag-based filters are specified in the config.
func getTags(ctx context.Context, client BackupVaultAPI, cfg config.ResourceType, backupVault types.BackupVaultListMember) (map[string]string, error) {
	tags := map[string]string{}
	if len(cfg.IncludeRule.Tags) > 0 || len(cfg.ExcludeRule.Tags) > 0 {
		tagsPaginator := backup.NewListTagsPaginator(client, &backup.ListTagsInput{
			ResourceArn: backupVault.BackupVaultArn,
		})
		for tagsPaginator.HasMorePages() {
			tagsPage, errListTags := tagsPaginator.NextPage(ctx)
			if errListTags != nil {
				return nil, errors.WithStackTrace(errListTags)
			}

			for tagKey, tagValue := range tagsPage.Tags {
				tags[tagKey] = tagValue
			}
		}
	}
	return tags, nil
}

// nukeRecoveryPoints deletes all recovery points in a backup vault.
func nukeRecoveryPoints(ctx context.Context, client BackupVaultAPI, name *string) error {
	vaultName := aws.ToString(name)
	logging.Debugf("Nuking the recovery points of backup vault %s", vaultName)

	// Use pagination to handle large numbers of recovery points
	paginator := backup.NewListRecoveryPointsByBackupVaultPaginator(client, &backup.ListRecoveryPointsByBackupVaultInput{
		BackupVaultName: name,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Failed] listing recovery points of backup vault %s: %v", vaultName, err)
			return err
		}

		for _, recoveryPoint := range page.RecoveryPoints {
			arn := aws.ToString(recoveryPoint.RecoveryPointArn)
			logging.Debugf("Deleting recovery point %s from backup vault %s", arn, vaultName)

			_, err = client.DeleteRecoveryPoint(ctx, &backup.DeleteRecoveryPointInput{
				BackupVaultName:  name,
				RecoveryPointArn: recoveryPoint.RecoveryPointArn,
			})
			if err != nil {
				logging.Debugf("[Failed] deleting recovery point %s: %v", arn, err)
				return err
			}
		}
	}

	// Wait until all recovery points are deleted
	if err := waitUntilRecoveryPointsDeleted(ctx, client, name); err != nil {
		logging.Debugf("[Failed] waiting for recovery points deletion in backup vault %s: %v", vaultName, err)
		return err
	}

	logging.Debugf("[OK] Successfully nuked recovery points of backup vault %s", vaultName)
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
			// Check if any recovery points still exist (only need first page with MaxResults=1)
			output, err := client.ListRecoveryPointsByBackupVault(ctx, &backup.ListRecoveryPointsByBackupVaultInput{
				BackupVaultName: name,
				MaxResults:      aws.Int32(1),
			})
			if err != nil {
				logging.Debugf("recovery point existence check error: %v", err)
				return err
			}

			if len(output.RecoveryPoints) == 0 {
				return nil
			}
			logging.Debugf("recovery point(s) still exist, waiting...")
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
