package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func getAllBackupVault(session *session.Session, configObj config.Config) ([]*string, error) {
	svc := backup.New(session)

	names := []*string{}
	paginator := func(output *backup.ListBackupVaultsOutput, lastPage bool) bool {
		for _, backupVault := range output.BackupVaultList {
			if shouldIncludeBackupVault(backupVault, configObj) {
				names = append(names, backupVault.BackupVaultName)
			}
		}

		return !lastPage
	}

	err := svc.ListBackupVaultsPages(&backup.ListBackupVaultsInput{}, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return names, nil
}

func shouldIncludeBackupVault(vault *backup.VaultListMember, configObj config.Config) bool {
	if vault == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(vault.BackupVaultName),
		configObj.BackupVault.IncludeRule.NamesRegExp,
		configObj.BackupVault.ExcludeRule.NamesRegExp,
	)
}

func nukeAllBackupVaults(session *session.Session, names []*string) error {
	svc := backup.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No backup vaults to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all backup vaults in region %s", *session.Config.Region)
	var deletedNames []*string

	for _, name := range names {
		_, err := svc.DeleteBackupVault(&backup.DeleteBackupVaultInput{
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
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking BackupVault",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted backup vault: %s", aws.StringValue(name))
		}
	}

	logging.Logger.Debugf("[OK] %d backup vault deleted in %s", len(deletedNames), *session.Config.Region)

	return nil
}
