package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type mockedSnapshot struct {
	ec2iface.EC2API
	DeleteSnapshotOutput    ec2.DeleteSnapshotOutput
	DescribeSnapshotsOutput ec2.DescribeSnapshotsOutput
}

func (m mockedSnapshot) DeleteSnapshot(input *ec2.DeleteSnapshotInput) (*ec2.DeleteSnapshotOutput, error) {
	return &m.DeleteSnapshotOutput, nil
}

func (m mockedSnapshot) DescribeSnapshots(input *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
	return &m.DescribeSnapshotsOutput, nil
}

func TestSnapshot_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := s.getAll(config.Config{
				Snapshots: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestSnapshot_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	s := Snapshots{
		Client: mockedSnapshot{
			DeleteSnapshotOutput: ec2.DeleteSnapshotOutput{},
		},
	}

	err := s.nukeAll([]*string{awsgo.String("test-snapshot")})
	require.NoError(t, err)
}
