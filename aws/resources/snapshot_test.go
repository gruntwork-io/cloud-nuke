package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockedSnapshot struct {
	SnapshotAPI
	DeleteSnapshotOutput    ec2.DeleteSnapshotOutput
	DescribeSnapshotsOutput ec2.DescribeSnapshotsOutput
	DescribeImagesOutput    ec2.DescribeImagesOutput
	DeregisterImageOutput   ec2.DeregisterImageOutput
}

func (m mockedSnapshot) DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error) {
	return &m.DeleteSnapshotOutput, nil
}

func (m mockedSnapshot) DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error) {
	return &m.DescribeSnapshotsOutput, nil
}
func (m mockedSnapshot) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOutput, nil
}
func (m mockedSnapshot) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	return &m.DeregisterImageOutput, nil
}

func TestSnapshot_GetAll(t *testing.T) {

	t.Parallel()

	testSnapshot1 := "test-snapshot1"
	testSnapshot2 := "test-snapshot2"
	now := time.Now()
	s := Snapshots{
		Client: mockedSnapshot{
			DescribeSnapshotsOutput: ec2.DescribeSnapshotsOutput{
				Snapshots: []types.Snapshot{
					{
						SnapshotId: aws.String(testSnapshot1),
						StartTime:  aws.Time(now),
						Tags: []types.Tag{
							{
								Key:   aws.String("aws:backup:source-resource"),
								Value: aws.String(""),
							},
						},
					},
					{
						SnapshotId: aws.String(testSnapshot2),
						StartTime:  aws.Time(now),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testSnapshot2},
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
			names, err := s.getAll(context.Background(), config.Config{
				Snapshots: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSnapshot_NukeAll(t *testing.T) {

	t.Parallel()

	s := Snapshots{
		Client: mockedSnapshot{
			DeleteSnapshotOutput: ec2.DeleteSnapshotOutput{},
		},
	}

	err := s.nukeAll([]*string{aws.String("test-snapshot")})
	require.NoError(t, err)
}
