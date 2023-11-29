package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
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
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking BackupVault",
			}, map[string]interface{}{
				"region": bv.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted backup vault: %s", aws.StringValue(name))
		}
	}

	logging.Debugf("[OK] %d backup vault deleted in %s", len(deletedNames), bv.Region)

	return nil
}
