package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/backup/backupiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedBackupVault struct {
	backupiface.BackupAPI
	ListBackupVaultsOutput                backup.ListBackupVaultsOutput
	ListRecoveryPointsByBackupVaultOutput backup.ListRecoveryPointsByBackupVaultOutput
	DeleteRecoveryPointOutput             backup.DeleteRecoveryPointOutput
	DeleteBackupVaultOutput               backup.DeleteBackupVaultOutput
}

func (m mockedBackupVault) ListBackupVaultsPagesWithContext(_ awsgo.Context, _ *backup.ListBackupVaultsInput, fn func(*backup.ListBackupVaultsOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListBackupVaultsOutput, true)
	return nil
}

func (m mockedBackupVault) DeleteBackupVaultWithContext(_ awsgo.Context, _ *backup.DeleteBackupVaultInput, _ ...request.Option) (*backup.DeleteBackupVaultOutput, error) {
	return &m.DeleteBackupVaultOutput, nil
}

func (m mockedBackupVault) ListRecoveryPointsByBackupVaultWithContext(aws.Context, *backup.ListRecoveryPointsByBackupVaultInput, ...request.Option) (*backup.ListRecoveryPointsByBackupVaultOutput, error) {
	return &m.ListRecoveryPointsByBackupVaultOutput, nil
}

func (m mockedBackupVault) WaitUntilRecoveryPointsDeleted(*string) error {
	return nil
}

func (m mockedBackupVault) DeleteRecoveryPointWithContext(aws.Context, *backup.DeleteRecoveryPointInput, ...request.Option) (*backup.DeleteRecoveryPointOutput, error) {
	return &m.DeleteRecoveryPointOutput, nil
}

func TestBackupVaultGetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-backup-vault-1"
	testName2 := "test-backup-vault-2"
	now := time.Now()
	bv := BackupVault{

		Client: mockedBackupVault{
			ListBackupVaultsOutput: backup.ListBackupVaultsOutput{
				BackupVaultList: []*backup.VaultListMember{
					{
						BackupVaultName: aws.String(testName1),
						CreationDate:    aws.Time(now),
					},
					{
						BackupVaultName: aws.String(testName2),
						CreationDate:    aws.Time(now.Add(1)),
					},
				}}},
	}
	bv.BaseAwsResource.Init(nil)

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := bv.getAll(context.Background(), config.Config{
				BackupVault: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, aws.StringValueSlice(names), tc.expected)
		})
	}
}

func TestBackupVaultNuke(t *testing.T) {

	t.Parallel()

	bv := BackupVault{
		Client: mockedBackupVault{
			DeleteBackupVaultOutput: backup.DeleteBackupVaultOutput{},
		},
	}
	bv.BaseAwsResource.Init(nil)
	bv.Context = context.Background()

	err := bv.nukeAll([]*string{aws.String("test-backup-vault")})
	require.NoError(t, err)
}
