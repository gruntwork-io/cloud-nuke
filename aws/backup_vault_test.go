package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupVaultNuke(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	backupVaultName := createBackupVault(t, session)
	err = nukeAllBackupVaults(session, []*string{backupVaultName})
	require.NoError(t, err)

	backupVaultNames, err := getAllBackupVault(session, config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, aws.StringValueSlice(backupVaultNames), aws.StringValue(backupVaultName))
}

func createBackupVault(t *testing.T, session *session.Session) *string {
	svc := backup.New(session)
	output, err := svc.CreateBackupVault(&backup.CreateBackupVaultInput{
		BackupVaultName: aws.String(fmt.Sprintf("test-backup-vault-%s", util.UniqueID())),
	})
	require.NoError(t, err)

	return output.BackupVaultName
}
