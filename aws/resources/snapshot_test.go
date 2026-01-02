package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockSnapshotsClient struct {
	// Paginated snapshots: each slice represents a page
	Pages              [][]types.Snapshot
	DescribeImagesOut  ec2.DescribeImagesOutput
	DeleteSnapshotErr  error
	DeregisterImageErr error
	currentPage        int
}

func (m *mockSnapshotsClient) DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error) {
	if m.currentPage >= len(m.Pages) {
		return &ec2.DescribeSnapshotsOutput{}, nil
	}

	page := m.Pages[m.currentPage]
	m.currentPage++

	var nextToken *string
	if m.currentPage < len(m.Pages) {
		nextToken = aws.String("next")
	}

	return &ec2.DescribeSnapshotsOutput{
		Snapshots: page,
		NextToken: nextToken,
	}, nil
}

func (m *mockSnapshotsClient) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOut, nil
}

func (m *mockSnapshotsClient) DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error) {
	return &ec2.DeleteSnapshotOutput{}, m.DeleteSnapshotErr
}

func (m *mockSnapshotsClient) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	return &ec2.DeregisterImageOutput{}, m.DeregisterImageErr
}

func TestListSnapshots(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := map[string]struct {
		pages    [][]types.Snapshot
		cfg      config.ResourceType
		expected []string
	}{
		"filters out AWS Backup snapshots": {
			pages: [][]types.Snapshot{{
				{SnapshotId: aws.String("snap-backup"), StartTime: aws.Time(now), Tags: []types.Tag{{Key: aws.String("aws:backup:source-resource"), Value: aws.String("")}}},
				{SnapshotId: aws.String("snap-regular"), StartTime: aws.Time(now)},
			}},
			cfg:      config.ResourceType{},
			expected: []string{"snap-regular"},
		},
		"applies time filter": {
			pages: [][]types.Snapshot{{
				{SnapshotId: aws.String("snap-1"), StartTime: aws.Time(now)},
			}},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1 * time.Hour))},
			},
			expected: []string{},
		},
		"handles pagination": {
			pages: [][]types.Snapshot{
				{{SnapshotId: aws.String("snap-page1"), StartTime: aws.Time(now)}},
				{{SnapshotId: aws.String("snap-page2"), StartTime: aws.Time(now)}},
			},
			cfg:      config.ResourceType{},
			expected: []string{"snap-page1", "snap-page2"},
		},
		"empty result": {
			pages:    [][]types.Snapshot{{}},
			cfg:      config.ResourceType{},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockSnapshotsClient{Pages: tc.pages}
			ids, err := listSnapshots(context.Background(), mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteSnapshot(t *testing.T) {
	t.Parallel()

	mock := &mockSnapshotsClient{}
	err := deleteSnapshot(context.Background(), mock, aws.String("snap-test"))
	require.NoError(t, err)
}

func TestDeregisterSnapshotAMIs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		images    []types.Image
		expectErr bool
	}{
		"deregisters associated AMIs": {
			images: []types.Image{
				{ImageId: aws.String("ami-1")},
				{ImageId: aws.String("ami-2")},
			},
			expectErr: false,
		},
		"no associated AMIs": {
			images:    []types.Image{},
			expectErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockSnapshotsClient{
				DescribeImagesOut: ec2.DescribeImagesOutput{Images: tc.images},
			}
			err := deregisterSnapshotAMIs(context.Background(), mock, aws.String("snap-test"))
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
