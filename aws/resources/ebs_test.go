package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

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

func (m mockedEBS) DeleteVolumeWithContext(_ awsgo.Context, input *ec2.DeleteVolumeInput, _ ...request.Option) (*ec2.DeleteVolumeOutput, error) {
	return &m.DeleteVolumeOutput, nil
}

func (m mockedEBS) DescribeVolumesWithContext(_ awsgo.Context, input *ec2.DescribeVolumesInput, _ ...request.Option) (*ec2.DescribeVolumesOutput, error) {
	return &m.DescribeVolumesOutput, nil
}

func (m mockedEBS) WaitUntilVolumeDeletedWithContext(_ awsgo.Context, input *ec2.DescribeVolumesInput, _ ...request.WaiterOption) error {
	return nil
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestEBSVolumne_NukeAll(t *testing.T) {

	t.Parallel()

	ev := EBSVolumes{
		Client: mockedEBS{
			DeleteVolumeOutput: ec2.DeleteVolumeOutput{},
		},
	}

	err := ev.nukeAll([]*string{aws.String("test-volume")})
	require.NoError(t, err)

}
