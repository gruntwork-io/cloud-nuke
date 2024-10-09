package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedBackupVault struct {
	BackupVaultAPI
	DeleteBackupVaultOutput               backup.DeleteBackupVaultOutput
	DeleteRecoveryPointOutput             backup.DeleteRecoveryPointOutput
	ListBackupVaultsOutput                backup.ListBackupVaultsOutput
	ListRecoveryPointsByBackupVaultOutput backup.ListRecoveryPointsByBackupVaultOutput
}

func (m mockedBackupVault) DeleteBackupVault(ctx context.Context, params *backup.DeleteBackupVaultInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupVaultOutput, error) {
	return &m.DeleteBackupVaultOutput, nil
}

func (m mockedBackupVault) DeleteRecoveryPoint(ctx context.Context, params *backup.DeleteRecoveryPointInput, optFns ...func(*backup.Options)) (*backup.DeleteRecoveryPointOutput, error) {
	return &m.DeleteRecoveryPointOutput, nil
}

func (m mockedBackupVault) ListBackupVaults(ctx context.Context, params *backup.ListBackupVaultsInput, optFns ...func(*backup.Options)) (*backup.ListBackupVaultsOutput, error) {
	return &m.ListBackupVaultsOutput, nil
}

func (m mockedBackupVault) ListRecoveryPointsByBackupVault(ctx context.Context, params *backup.ListRecoveryPointsByBackupVaultInput, optFns ...func(*backup.Options)) (*backup.ListRecoveryPointsByBackupVaultOutput, error) {
	return &m.ListRecoveryPointsByBackupVaultOutput, nil
}

func TestBackupVaultGetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-backup-vault-1"
	testName2 := "test-backup-vault-2"
	now := time.Now()
	bv := BackupVault{

		Client: mockedBackupVault{
			ListBackupVaultsOutput: backup.ListBackupVaultsOutput{
				BackupVaultList: []types.BackupVaultListMember{
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
			require.Equal(t, aws.ToStringSlice(names), tc.expected)
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
