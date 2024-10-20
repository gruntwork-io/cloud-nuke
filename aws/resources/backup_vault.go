package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (bv *BackupVault) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var names []*string
	paginator := backup.NewListBackupVaultsPaginator(bv.Client, &backup.ListBackupVaultsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, backupVault := range page.BackupVaultList {
			if configObj.BackupVault.ShouldInclude(config.ResourceValue{
				Name: backupVault.BackupVaultName,
				Time: backupVault.CreationDate,
			}) {
				names = append(names, backupVault.BackupVaultName)
			}
		}

	}

	return names, nil
}

func (bv *BackupVault) nuke(name *string) error {
	if err := bv.nukeRecoveryPoints(name); err != nil {
		return errors.WithStackTrace(err)
	}

	if err := bv.nukeBackupVault(name); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (bv *BackupVault) nukeBackupVault(name *string) error {
	_, err := bv.Client.DeleteBackupVault(bv.Context, &backup.DeleteBackupVaultInput{
		BackupVaultName: name,
	})
	if err != nil {
		logging.Debugf("[Failed] nuking the backup vault %s: %v", aws.ToString(name), err)
		return errors.WithStackTrace(err)
	}
	return nil
}

func (bv *BackupVault) nukeRecoveryPoints(name *string) error {
	logging.Debugf("Nuking the recovery points of backup vault %s", aws.ToString(name))

	output, err := bv.Client.ListRecoveryPointsByBackupVault(bv.Context, &backup.ListRecoveryPointsByBackupVaultInput{
		BackupVaultName: name,
	})

	if err != nil {
		logging.Debugf("[Failed] listing the recovery points of backup vault %s: %v", aws.ToString(name), err)
		return errors.WithStackTrace(err)
	}

	for _, recoveryPoint := range output.RecoveryPoints {
		logging.Debugf("Deleting recovery point %s from backup vault %s", aws.ToString(recoveryPoint.RecoveryPointArn), aws.ToString(name))
		_, err := bv.Client.DeleteRecoveryPoint(bv.Context, &backup.DeleteRecoveryPointInput{
			BackupVaultName:  name,
			RecoveryPointArn: recoveryPoint.RecoveryPointArn,
		})

		if err != nil {
			logging.Debugf("[Failed] nuking the backup vault %s: %v", aws.ToString(name), err)
			return errors.WithStackTrace(err)
		}
	}

	// wait until all the recovery points nuked successfully
	err = bv.WaitUntilRecoveryPointsDeleted(name)
	if err != nil {
		logging.Debugf("[Failed] waiting deletion of recovery points for backup vault %s: %v", aws.ToString(name), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[Ok] successfully nuked recovery points of backup vault %s", aws.ToString(name))

	return nil
}

func (bv *BackupVault) WaitUntilRecoveryPointsDeleted(name *string) error {
	timeoutCtx, cancel := context.WithTimeout(bv.Context, 1*time.Minute)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("recovery point deletion check timed out after 1 minute")
		case <-ticker.C:
			output, err := bv.Client.ListRecoveryPointsByBackupVault(bv.Context, &backup.ListRecoveryPointsByBackupVaultInput{
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

func (bv *BackupVault) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No backup vaults to nuke in region %s", bv.Region)
		return nil
	}

	logging.Debugf("Deleting all backup vaults in region %s", bv.Region)
	var deletedNames []*string

	for _, name := range names {
		err := bv.nuke(name)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "Backup Vault",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted backup vault: %s", aws.ToString(name))
		}
	}

	logging.Debugf("[OK] %d backup vault deleted in %s", len(deletedNames), bv.Region)

	return nil
}
