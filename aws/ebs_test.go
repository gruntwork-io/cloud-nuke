package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEBS struct {
	ec2iface.EC2API
	DeleteVolumeOutput    ec2.DeleteVolumeOutput
	DescribeVolumesOutput ec2.DescribeVolumesOutput
}

func (m mockedEBS) DeleteVolume(input *ec2.DeleteVolumeInput) (*ec2.DeleteVolumeOutput, error) {
	return &m.DeleteVolumeOutput, nil
}

func (m mockedEBS) DescribeVolumes(input *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error) {
	return &m.DescribeVolumesOutput, nil
}

func (m mockedEBS) WaitUntilVolumeDeleted(input *ec2.DescribeVolumesInput) error {
	return nil
}

func TestEBSVolume_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	testVolume1 := "test-volume1"
	testVolume2 := "test-volume2"
	now := time.Now()
	ev := EBSVolumes{
		Client: mockedEBS{
			DescribeVolumesOutput: ec2.DescribeVolumesOutput{
				Volumes: []*ec2.Volume{
					{
						VolumeId:   awsgo.String(testVolume1),
						CreateTime: awsgo.Time(now),
						Tags: []*ec2.Tag{{
							Key:   awsgo.String("Name"),
							Value: awsgo.String(testName1),
						}},
					},
					{
						VolumeId:   awsgo.String(testVolume2),
						CreateTime: awsgo.Time(now.Add(1)),
						Tags: []*ec2.Tag{{
							Key:   awsgo.String("Name"),
							Value: awsgo.String(testName2),
						}},
					},
				}}}}

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
					TimeAfter: aws.Time(now.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ev.getAll(config.Config{
				EBSVolume: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestEBSVolumne_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ev := EBSVolumes{
		Client: mockedEBS{
			DeleteVolumeOutput: ec2.DeleteVolumeOutput{},
		},
	}

	err := ev.nukeAll([]*string{aws.String("test-volume")})
	require.NoError(t, err)

}
