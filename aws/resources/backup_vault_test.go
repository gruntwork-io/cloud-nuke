package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/service/backup/backupiface"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedBackupVault struct {
	backupiface.BackupAPI
	ListBackupVaultsOutput  backup.ListBackupVaultsOutput
	DeleteBackupVaultOutput backup.DeleteBackupVaultOutput
}

func (m mockedBackupVault) ListBackupVaultsPages(
	input *backup.ListBackupVaultsInput, fn func(*backup.ListBackupVaultsOutput, bool) bool) error {
	fn(&m.ListBackupVaultsOutput, true)
	return nil
}

func (m mockedBackupVault) DeleteBackupVault(*backup.DeleteBackupVaultInput) (*backup.DeleteBackupVaultOutput, error) {
	return &m.DeleteBackupVaultOutput, nil
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

	err := bv.nukeAll([]*string{aws.String("test-backup-vault")})
	require.NoError(t, err)
}
