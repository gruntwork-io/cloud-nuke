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
	"github.com/stretchr/testify/require"
)

type mockedEBS struct {
	EBSVolumesAPI
	DescribeVolumesOutput ec2.DescribeVolumesOutput
	DeleteVolumeOutput    ec2.DeleteVolumeOutput
}

func (m mockedEBS) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return &m.DescribeVolumesOutput, nil
}

func (m mockedEBS) DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error) {
	return &m.DeleteVolumeOutput, nil
}

func TestEBSVolume_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testVolume1 := "test-volume1"
	testVolume2 := "test-volume2"
	now := time.Now()
	ev := EBSVolumes{
		Client: mockedEBS{
			DescribeVolumesOutput: ec2.DescribeVolumesOutput{
				Volumes: []types.Volume{
					{
						VolumeId:   aws.String(testVolume1),
						CreateTime: aws.Time(now),
						Tags: []types.Tag{{
							Key:   aws.String("Name"),
							Value: aws.String(testName1),
						}},
					},
					{
						VolumeId:   aws.String(testVolume2),
						CreateTime: aws.Time(now.Add(1)),
						Tags: []types.Tag{{
							Key:   aws.String("Name"),
							Value: aws.String(testName2),
						}},
					},
				}}}}

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
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testVolume2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ev.getAll(context.Background(), config.Config{
				EBSVolume: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEBSVolume_NukeAll(t *testing.T) {
	t.Parallel()

	ev := EBSVolumes{
		Client: mockedEBS{
			DeleteVolumeOutput: ec2.DeleteVolumeOutput{},
		},
	}

	err := ev.nukeAll([]*string{aws.String("test-volume")})
	require.NoError(t, err)
}
