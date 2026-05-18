package resources

import (
	"context"
	"errors"
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

type mockBackupPlanListTagsPageOutput struct {
	ListTagsPages []backup.ListTagsOutput
	listTagsIndex int
}

type mockBackupPlanClient struct {
	// ListBackupPlans pagination support
	ListBackupPlansPages []backup.ListBackupPlansOutput
	listPlansPageIndex   int

	// ListBackupSelections pagination support
	ListBackupSelectionsPages  []backup.ListBackupSelectionsOutput
	listSelectionsPageIndex    int
	DeleteBackupSelectionCount atomic.Int32

	// ListTags pagination support
	ListTagsPages map[string]*mockBackupPlanListTagsPageOutput
	ListTagsErr   error
}

func (m *mockBackupPlanClient) DeleteBackupPlan(ctx context.Context, params *backup.DeleteBackupPlanInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupPlanOutput, error) {
	return &backup.DeleteBackupPlanOutput{}, nil
}

func (m *mockBackupPlanClient) DeleteBackupSelection(ctx context.Context, params *backup.DeleteBackupSelectionInput, optFns ...func(*backup.Options)) (*backup.DeleteBackupSelectionOutput, error) {
	m.DeleteBackupSelectionCount.Add(1)
	return &backup.DeleteBackupSelectionOutput{}, nil
}

func (m *mockBackupPlanClient) ListBackupPlans(ctx context.Context, params *backup.ListBackupPlansInput, optFns ...func(*backup.Options)) (*backup.ListBackupPlansOutput, error) {
	if m.listPlansPageIndex >= len(m.ListBackupPlansPages) {
		return &backup.ListBackupPlansOutput{}, nil
	}
	output := m.ListBackupPlansPages[m.listPlansPageIndex]
	m.listPlansPageIndex++
	return &output, nil
}

func (m *mockBackupPlanClient) ListBackupSelections(ctx context.Context, params *backup.ListBackupSelectionsInput, optFns ...func(*backup.Options)) (*backup.ListBackupSelectionsOutput, error) {
	if m.listSelectionsPageIndex >= len(m.ListBackupSelectionsPages) {
		return &backup.ListBackupSelectionsOutput{}, nil
	}
	output := m.ListBackupSelectionsPages[m.listSelectionsPageIndex]
	m.listSelectionsPageIndex++
	return &output, nil
}

func (m *mockBackupPlanClient) ListTags(ctx context.Context, params *backup.ListTagsInput, optFns ...func(*backup.Options)) (*backup.ListTagsOutput, error) {
	if m.ListTagsErr != nil {
		return nil, m.ListTagsErr
	}
	pages := m.ListTagsPages[*params.ResourceArn]
	if pages == nil || pages.listTagsIndex >= len(pages.ListTagsPages) {
		return &backup.ListTagsOutput{}, nil
	}

	output := pages.ListTagsPages[pages.listTagsIndex]
	pages.listTagsIndex++

	if pages.listTagsIndex < len(pages.ListTagsPages) {
		output.NextToken = aws.String("token")
	}

	return &output, nil
}

func TestListBackupPlans(t *testing.T) {
	t.Parallel()

	testId1 := "plan-id-1"
	testId2 := "plan-id-2"
	now := time.Now()

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test-plan-1"),
					}}},
			},
			expected: []string{testId2},
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
			mock := &mockBackupPlanClient{
				ListBackupPlansPages: []backup.ListBackupPlansOutput{
					{
						BackupPlansList: []types.BackupPlansListMember{
							{BackupPlanId: aws.String(testId1), BackupPlanName: aws.String("test-plan-1"), CreationDate: aws.Time(now)},
							{BackupPlanId: aws.String(testId2), BackupPlanName: aws.String("test-plan-2"), CreationDate: aws.Time(now.Add(1))},
						},
					},
				},
			}

			ids, err := listBackupPlans(context.Background(), mock, resource.Scope{}, tc.configObj)

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestListBackupPlans_FilterTags(t *testing.T) {
	t.Parallel()

	planWithoutTags := "plan-no-tags"
	planFooFaz := "plan-foo-faz"
	planFoo := "plan-foo"

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{planWithoutTags, planFooFaz, planFoo},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				}},
			expected: []string{planFooFaz, planFoo},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"faz": {RE: *regexp.MustCompile("baz")}},
				}},
			expected: []string{planWithoutTags, planFoo},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockBackupPlanClient{
				ListBackupPlansPages: []backup.ListBackupPlansOutput{
					{
						BackupPlansList: []types.BackupPlansListMember{
							{BackupPlanId: aws.String(planWithoutTags), BackupPlanName: aws.String("no-tags"), BackupPlanArn: aws.String("arn:aws:" + planWithoutTags)},
							{BackupPlanId: aws.String(planFooFaz), BackupPlanName: aws.String("foo-faz"), BackupPlanArn: aws.String("arn:aws:" + planFooFaz)},
							{BackupPlanId: aws.String(planFoo), BackupPlanName: aws.String("foo"), BackupPlanArn: aws.String("arn:aws:" + planFoo)},
						},
					},
				},
				ListTagsPages: map[string]*mockBackupPlanListTagsPageOutput{
					"arn:aws:" + planFoo: {
						ListTagsPages: []backup.ListTagsOutput{
							{Tags: map[string]string{"foo": "bar"}},
						},
					},
					"arn:aws:" + planFooFaz: {
						ListTagsPages: []backup.ListTagsOutput{
							{Tags: map[string]string{"foo": "bar"}},
							{Tags: map[string]string{"faz": "baz"}},
						},
					},
				},
			}

			ids, err := listBackupPlans(context.Background(), mock, resource.Scope{}, tc.configObj)

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestListBackupPlans_TagsError(t *testing.T) {
	t.Parallel()

	testArn := "arn:aws:backup:us-east-1:123456789012:backup-plan:plan-1"
	mock := &mockBackupPlanClient{
		ListBackupPlansPages: []backup.ListBackupPlansOutput{
			{
				BackupPlansList: []types.BackupPlansListMember{
					{BackupPlanId: aws.String("plan-1"), BackupPlanName: aws.String("test-plan"), BackupPlanArn: aws.String(testArn)},
				},
			},
		},
		ListTagsErr: errors.New("BackupPlanException"),
	}

	cfg := config.ResourceType{
		IncludeRule: config.FilterRule{
			Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
		},
	}

	ids, err := listBackupPlans(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{}, aws.ToStringSlice(ids))
}

func TestNukeBackupPlan(t *testing.T) {
	t.Parallel()

	mock := &mockBackupPlanClient{}

	err := nukeBackupPlan(context.Background(), mock, aws.String("plan-1"))
	require.NoError(t, err)
}

func TestNukeBackupSelections(t *testing.T) {
	t.Parallel()

	t.Run("no selections", func(t *testing.T) {
		mock := &mockBackupPlanClient{
			ListBackupSelectionsPages: []backup.ListBackupSelectionsOutput{
				{BackupSelectionsList: []types.BackupSelectionsListMember{}},
			},
		}

		err := nukeBackupSelections(context.Background(), mock, aws.String("plan-1"))
		require.NoError(t, err)
		require.Equal(t, int32(0), mock.DeleteBackupSelectionCount.Load())
	})

	t.Run("with selections", func(t *testing.T) {
		mock := &mockBackupPlanClient{
			ListBackupSelectionsPages: []backup.ListBackupSelectionsOutput{
				{
					BackupSelectionsList: []types.BackupSelectionsListMember{
						{SelectionId: aws.String("sel-1"), BackupPlanId: aws.String("plan-1")},
						{SelectionId: aws.String("sel-2"), BackupPlanId: aws.String("plan-1")},
					},
				},
			},
		}

		err := nukeBackupSelections(context.Background(), mock, aws.String("plan-1"))
		require.NoError(t, err)
		require.Equal(t, int32(2), mock.DeleteBackupSelectionCount.Load())
	})
}

func TestListBackupPlansPagination(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockBackupPlanClient{
		ListBackupPlansPages: []backup.ListBackupPlansOutput{
			{
				BackupPlansList: []types.BackupPlansListMember{
					{BackupPlanId: aws.String("plan-1"), BackupPlanName: aws.String("plan-1"), CreationDate: aws.Time(now)},
				},
				NextToken: aws.String("token1"),
			},
			{
				BackupPlansList: []types.BackupPlansListMember{
					{BackupPlanId: aws.String("plan-2"), BackupPlanName: aws.String("plan-2"), CreationDate: aws.Time(now)},
				},
			},
		},
	}

	ids, err := listBackupPlans(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"plan-1", "plan-2"}, aws.ToStringSlice(ids))
}
