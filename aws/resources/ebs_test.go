package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockEBSVolumesClient struct {
	DescribeVolumesOutput ec2.DescribeVolumesOutput
	DeleteVolumeOutput    ec2.DeleteVolumeOutput
}

func (m *mockEBSVolumesClient) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return &m.DescribeVolumesOutput, nil
}

func (m *mockEBSVolumesClient) DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error) {
	return &m.DeleteVolumeOutput, nil
}

func TestListEBSVolumes(t *testing.T) {
	t.Parallel()

	testVolume1 := "vol-test1"
	testVolume2 := "vol-test2"
	now := time.Now()

	mock := &mockEBSVolumesClient{
		DescribeVolumesOutput: ec2.DescribeVolumesOutput{
			Volumes: []types.Volume{
				{
					VolumeId:   aws.String(testVolume1),
					CreateTime: aws.Time(now),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String("test-name1"),
					}},
				},
				{
					VolumeId:   aws.String(testVolume2),
					CreateTime: aws.Time(now.Add(1 * time.Hour)),
					Tags: []types.Tag{{
						Key:   aws.String("Name"),
						Value: aws.String("test-name2"),
					}},
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
			expected:  []string{testVolume1, testVolume2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-name1")}},
				},
			},
			expected: []string{testVolume2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listEBSVolumes(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteEBSVolume(t *testing.T) {
	t.Parallel()

	mock := &mockEBSVolumesClient{
		DeleteVolumeOutput: ec2.DeleteVolumeOutput{},
	}

	err := deleteEBSVolume(context.Background(), mock, aws.String("vol-test"))
	require.NoError(t, err)
}
