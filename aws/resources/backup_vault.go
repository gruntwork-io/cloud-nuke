package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (bv *BackupVault) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	names := []*string{}
	paginator := func(output *backup.ListBackupVaultsOutput, lastPage bool) bool {
		for _, backupVault := range output.BackupVaultList {
			if configObj.BackupVault.ShouldInclude(config.ResourceValue{
				Name: backupVault.BackupVaultName,
				Time: backupVault.CreationDate,
			}) {
				names = append(names, backupVault.BackupVaultName)
			}
		}

		return !lastPage
	}

	err := bv.Client.ListBackupVaultsPages(&backup.ListBackupVaultsInput{}, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return names, nil
}

func (bv *BackupVault) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No backup vaults to nuke in region %s", bv.Region)
		return nil
	}

	logging.Debugf("Deleting all backup vaults in region %s", bv.Region)
	var deletedNames []*string

	for _, name := range names {
		_, err := bv.Client.DeleteBackupVault(&backup.DeleteBackupVaultInput{
			BackupVaultName: name,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "Backup Vault",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted backup vault: %s", aws.StringValue(name))
		}
	}

	logging.Debugf("[OK] %d backup vault deleted in %s", len(deletedNames), bv.Region)

	return nil
}
