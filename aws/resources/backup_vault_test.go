package resources

import (
	"context"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockListTagsPageOutput struct {
	ListTagsPages []backup.ListTagsOutput
	listTagsIndex int
}

type mockBackupVaultClient struct {
	// ListBackupVaults pagination support
	ListBackupVaultsPages []backup.ListBackupVaultsOutput
	listVaultsPageIndex   int

	// ListRecoveryPointsByBackupVault pagination support
	ListRecoveryPointsPages  []backup.ListRecoveryPointsByBackupVaultOutput
	listRecoveryPointsIndex  int
	DeleteRecoveryPointCount atomic.Int32

	//ListTags pagination support
	ListTagsPages map[string]*mockListTagsPageOutput
}

func (m *mockBackupVaultClient) DeleteBackupVault(ctx context.Context, params *backup.DeleteBackupVaultInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupVaultOutput, error) {
	return &backup.DeleteBackupVaultOutput{}, nil
}

func (m *mockBackupVaultClient) DeleteRecoveryPoint(ctx context.Context, params *backup.DeleteRecoveryPointInput, optFns ...func(*backup.Options)) (*backup.DeleteRecoveryPointOutput, error) {
	m.DeleteRecoveryPointCount.Add(1)
	return &backup.DeleteRecoveryPointOutput{}, nil
}

func (m *mockBackupVaultClient) ListBackupVaults(ctx context.Context, params *backup.ListBackupVaultsInput, optFns ...func(*backup.Options)) (*backup.ListBackupVaultsOutput, error) {
	if m.listVaultsPageIndex >= len(m.ListBackupVaultsPages) {
		return &backup.ListBackupVaultsOutput{}, nil
	}
	output := m.ListBackupVaultsPages[m.listVaultsPageIndex]
	m.listVaultsPageIndex++
	return &output, nil
}

func (m *mockBackupVaultClient) ListRecoveryPointsByBackupVault(ctx context.Context, params *backup.ListRecoveryPointsByBackupVaultInput, optFns ...func(*backup.Options)) (*backup.ListRecoveryPointsByBackupVaultOutput, error) {
	if m.listRecoveryPointsIndex >= len(m.ListRecoveryPointsPages) {
		return &backup.ListRecoveryPointsByBackupVaultOutput{}, nil
	}
	output := m.ListRecoveryPointsPages[m.listRecoveryPointsIndex]
	m.listRecoveryPointsIndex++
	return &output, nil
}

func (m *mockBackupVaultClient) ListTags(ctx context.Context, params *backup.ListTagsInput, optFns ...func(*backup.Options)) (*backup.ListTagsOutput, error) {
	pages := m.ListTagsPages[*params.ResourceArn]
	if pages == nil || pages.listTagsIndex >= len(pages.ListTagsPages) {
		return &backup.ListTagsOutput{}, nil
	}

	output := pages.ListTagsPages[pages.listTagsIndex]
	pages.listTagsIndex++

	// Set NextToken if there are more pages to indicate pagination should continue
	if pages.listTagsIndex < len(pages.ListTagsPages) {
		output.NextToken = aws.String("token")
	}

	return &output, nil
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
				ListBackupVaultsPages: []backup.ListBackupVaultsOutput{
					{
						BackupVaultList: []types.BackupVaultListMember{
							{BackupVaultName: aws.String(testName1), CreationDate: aws.Time(now)},
							{BackupVaultName: aws.String(testName2), CreationDate: aws.Time(now.Add(1))},
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

func TestListBackupVaults_FilterTags(t *testing.T) {
	t.Parallel()

	backupVaultWithoutTags := "backupVaultWithoutTags"
	backVaultFooFaz := "backupVaultFooFaz"
	backVaultFoo := "backupVaultFoo"

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{backupVaultWithoutTags, backVaultFooFaz, backVaultFoo},
		},
		"tagInclusionFiler": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				}},
			expected: []string{backVaultFooFaz, backVaultFoo},
		},
		"tagExclusionFiler": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"faz": {RE: *regexp.MustCompile("baz")}},
				}},
			expected: []string{backupVaultWithoutTags, backVaultFoo},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a fresh mock for each subtest to avoid state pollution
			mock := &mockBackupVaultClient{
				ListBackupVaultsPages: []backup.ListBackupVaultsOutput{
					{
						BackupVaultList: []types.BackupVaultListMember{
							{BackupVaultName: aws.String(backupVaultWithoutTags), BackupVaultArn: aws.String("arn:aws:" + backupVaultWithoutTags)},
							{BackupVaultName: aws.String(backVaultFooFaz), BackupVaultArn: aws.String("arn:aws:" + backVaultFooFaz)},
							{BackupVaultName: aws.String(backVaultFoo), BackupVaultArn: aws.String("arn:aws:" + backVaultFoo)},
						},
					},
				},
				ListTagsPages: map[string]*mockListTagsPageOutput{
					"arn:aws:" + backVaultFoo: {
						ListTagsPages: []backup.ListTagsOutput{
							{Tags: map[string]string{"foo": "bar"}},
						},
					},
					"arn:aws:" + backVaultFooFaz: {
						ListTagsPages: []backup.ListTagsOutput{
							{Tags: map[string]string{"foo": "bar"}},
							{Tags: map[string]string{"faz": "baz"}},
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

	mock := &mockBackupVaultClient{}

	err := nukeBackupVault(context.Background(), mock, aws.String("test-backup-vault"))
	require.NoError(t, err)
}

func TestNukeRecoveryPoints(t *testing.T) {
	t.Parallel()

	t.Run("no recovery points", func(t *testing.T) {
		mock := &mockBackupVaultClient{
			ListRecoveryPointsPages: []backup.ListRecoveryPointsByBackupVaultOutput{
				{RecoveryPoints: []types.RecoveryPointByBackupVault{}},
			},
		}

		err := nukeRecoveryPoints(context.Background(), mock, aws.String("test-backup-vault"))
		require.NoError(t, err)
		require.Equal(t, int32(0), mock.DeleteRecoveryPointCount.Load())
	})

	t.Run("with recovery points", func(t *testing.T) {
		mock := &mockBackupVaultClient{
			ListRecoveryPointsPages: []backup.ListRecoveryPointsByBackupVaultOutput{
				{
					RecoveryPoints: []types.RecoveryPointByBackupVault{
						{RecoveryPointArn: aws.String("arn:aws:backup:us-east-1:123456789012:recovery-point:rp-1")},
						{RecoveryPointArn: aws.String("arn:aws:backup:us-east-1:123456789012:recovery-point:rp-2")},
					},
				},
				// Second call returns empty (for waitUntilRecoveryPointsDeleted)
				{RecoveryPoints: []types.RecoveryPointByBackupVault{}},
			},
		}

		err := nukeRecoveryPoints(context.Background(), mock, aws.String("test-backup-vault"))
		require.NoError(t, err)
		require.Equal(t, int32(2), mock.DeleteRecoveryPointCount.Load())
	})
}

func TestListBackupVaultsPagination(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockBackupVaultClient{
		ListBackupVaultsPages: []backup.ListBackupVaultsOutput{
			{
				BackupVaultList: []types.BackupVaultListMember{
					{BackupVaultName: aws.String("vault-1"), CreationDate: aws.Time(now)},
				},
				NextToken: aws.String("token1"),
			},
			{
				BackupVaultList: []types.BackupVaultListMember{
					{BackupVaultName: aws.String("vault-2"), CreationDate: aws.Time(now)},
				},
			},
		},
	}

	names, err := listBackupVaults(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"vault-1", "vault-2"}, aws.ToStringSlice(names))
}
