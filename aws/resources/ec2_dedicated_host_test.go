package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEC2DedicatedHosts struct {
	ec2iface.EC2API
	DescribeHostsPagesOutput ec2.DescribeHostsOutput
	ReleaseHostsOutput       ec2.ReleaseHostsOutput
}

func (m mockedEC2DedicatedHosts) DescribeHostsPagesWithContext(_ awsgo.Context, _ *ec2.DescribeHostsInput, fn func(*ec2.DescribeHostsOutput, bool) bool, _ ...request.Option) error {
	fn(&m.DescribeHostsPagesOutput, true)
	return nil
}

func (m mockedEC2DedicatedHosts) ReleaseHostsWithContext(_ awsgo.Context, _ *ec2.ReleaseHostsInput, _ ...request.Option) (*ec2.ReleaseHostsOutput, error) {
	return &m.ReleaseHostsOutput, nil
}

func TestEC2DedicatedHosts_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "test-host-id-1"
	testId2 := "test-host-id-2"
	testName1 := "test-host-name-1"
	testName2 := "test-host-name-2"
	now := time.Now()
	h := EC2DedicatedHosts{
		Client: mockedEC2DedicatedHosts{
			DescribeHostsPagesOutput: ec2.DescribeHostsOutput{
				Hosts: []*ec2.Host{
					{
						HostId: awsgo.String(testId1),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
						},
						AllocationTime: awsgo.Time(now),
					},
					{
						HostId: awsgo.String(testId2),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
						},
						AllocationTime: awsgo.Time(now.Add(1)),
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
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := h.getAll(context.Background(), config.Config{
				EC2DedicatedHosts: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestEC2DedicatedHosts_NukeAll(t *testing.T) {

	t.Parallel()

	h := EC2DedicatedHosts{
		Client: mockedEC2DedicatedHosts{
			ReleaseHostsOutput: ec2.ReleaseHostsOutput{},
		},
	}

	err := h.nukeAll([]*string{awsgo.String("test-host-id-1"), awsgo.String("test-host-id-2")})
	require.NoError(t, err)
}
