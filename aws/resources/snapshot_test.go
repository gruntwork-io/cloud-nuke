package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type mockedSnapshot struct {
	ec2iface.EC2API
	DeleteSnapshotOutput    ec2.DeleteSnapshotOutput
	DescribeSnapshotsOutput ec2.DescribeSnapshotsOutput
	DescribeImagesOutput    ec2.DescribeImagesOutput
	DeregisterImageOutput   ec2.DeregisterImageOutput
}

func (m mockedSnapshot) DeleteSnapshotWithContext(_ awsgo.Context, _ *ec2.DeleteSnapshotInput, _ ...request.Option) (*ec2.DeleteSnapshotOutput, error) {
	return &m.DeleteSnapshotOutput, nil
}

func (m mockedSnapshot) DescribeSnapshotsWithContext(_ awsgo.Context, _ *ec2.DescribeSnapshotsInput, _ ...request.Option) (*ec2.DescribeSnapshotsOutput, error) {
	return &m.DescribeSnapshotsOutput, nil
}
func (m mockedSnapshot) DescribeImagesWithContext(awsgo.Context, *ec2.DescribeImagesInput, ...request.Option) (*ec2.DescribeImagesOutput, error) {
	return &m.DescribeImagesOutput, nil
}
func (m mockedSnapshot) DeregisterImageWithContext(awsgo.Context, *ec2.DeregisterImageInput, ...request.Option) (*ec2.DeregisterImageOutput, error) {
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
				Snapshots: []*ec2.Snapshot{
					{
						SnapshotId: awsgo.String(testSnapshot1),
						StartTime:  awsgo.Time(now),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("aws:backup:source-resource"),
								Value: awsgo.String(""),
							},
						},
					},
					{
						SnapshotId: awsgo.String(testSnapshot2),
						StartTime:  awsgo.Time(now),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
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

	err := s.nukeAll([]*string{awsgo.String("test-snapshot")})
	require.NoError(t, err)
}
