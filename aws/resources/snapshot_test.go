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
	DescribeSnapshotsOutput ec2.DescribeSnapshotsOutput
	DescribeImagesOutput    ec2.DescribeImagesOutput
	DeleteSnapshotOutput    ec2.DeleteSnapshotOutput
	DeregisterImageOutput   ec2.DeregisterImageOutput
}

func (m *mockSnapshotsClient) DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error) {
	return &m.DescribeSnapshotsOutput, nil
}

func (m *mockSnapshotsClient) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOutput, nil
}

func (m *mockSnapshotsClient) DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error) {
	return &m.DeleteSnapshotOutput, nil
}

func (m *mockSnapshotsClient) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	return &m.DeregisterImageOutput, nil
}

func TestListSnapshots(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := map[string]struct {
		snapshots []types.Snapshot
		cfg       config.ResourceType
		expected  []string
	}{
		"filters out AWS Backup snapshots": {
			snapshots: []types.Snapshot{
				{SnapshotId: aws.String("snap-backup"), StartTime: aws.Time(now), Tags: []types.Tag{{Key: aws.String("aws:backup:source-resource"), Value: aws.String("")}}},
				{SnapshotId: aws.String("snap-regular"), StartTime: aws.Time(now)},
			},
			cfg:      config.ResourceType{},
			expected: []string{"snap-regular"},
		},
		"applies time filter": {
			snapshots: []types.Snapshot{
				{SnapshotId: aws.String("snap-1"), StartTime: aws.Time(now)},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1 * time.Hour))},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockSnapshotsClient{
				DescribeSnapshotsOutput: ec2.DescribeSnapshotsOutput{Snapshots: tc.snapshots},
			}
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
