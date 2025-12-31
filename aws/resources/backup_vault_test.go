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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockBackupVaultClient struct {
	DeleteBackupVaultOutput               backup.DeleteBackupVaultOutput
	DeleteRecoveryPointOutput             backup.DeleteRecoveryPointOutput
	ListBackupVaultsOutput                backup.ListBackupVaultsOutput
	ListRecoveryPointsByBackupVaultOutput backup.ListRecoveryPointsByBackupVaultOutput
}

func (m *mockBackupVaultClient) DeleteBackupVault(ctx context.Context, params *backup.DeleteBackupVaultInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupVaultOutput, error) {
	return &m.DeleteBackupVaultOutput, nil
}

func (m *mockBackupVaultClient) DeleteRecoveryPoint(ctx context.Context, params *backup.DeleteRecoveryPointInput, optFns ...func(*backup.Options)) (*backup.DeleteRecoveryPointOutput, error) {
	return &m.DeleteRecoveryPointOutput, nil
}

func (m *mockBackupVaultClient) ListBackupVaults(ctx context.Context, params *backup.ListBackupVaultsInput, optFns ...func(*backup.Options)) (*backup.ListBackupVaultsOutput, error) {
	return &m.ListBackupVaultsOutput, nil
}

func (m *mockBackupVaultClient) ListRecoveryPointsByBackupVault(ctx context.Context, params *backup.ListRecoveryPointsByBackupVaultInput, optFns ...func(*backup.Options)) (*backup.ListRecoveryPointsByBackupVaultOutput, error) {
	return &m.ListRecoveryPointsByBackupVaultOutput, nil
}

func TestListBackupVaults(t *testing.T) {
	t.Parallel()

	testName1 := "test-backup-vault-1"
	testName2 := "test-backup-vault-2"
	now := time.Now()

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
			mock := &mockBackupVaultClient{
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
					},
				},
			}

			names, err := listBackupVaults(context.Background(), mock, resource.Scope{}, tc.configObj)

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestNukeBackupVault(t *testing.T) {
	t.Parallel()

	mock := &mockBackupVaultClient{
		DeleteBackupVaultOutput:               backup.DeleteBackupVaultOutput{},
		ListRecoveryPointsByBackupVaultOutput: backup.ListRecoveryPointsByBackupVaultOutput{},
	}

	err := nukeBackupVault(context.Background(), mock, aws.String("test-backup-vault"))
	require.NoError(t, err)
}

func TestNukeRecoveryPoints(t *testing.T) {
	t.Parallel()

	mock := &mockBackupVaultClient{
		DeleteRecoveryPointOutput:             backup.DeleteRecoveryPointOutput{},
		ListRecoveryPointsByBackupVaultOutput: backup.ListRecoveryPointsByBackupVaultOutput{},
	}

	err := nukeRecoveryPoints(context.Background(), mock, aws.String("test-backup-vault"))
	require.NoError(t, err)
}
