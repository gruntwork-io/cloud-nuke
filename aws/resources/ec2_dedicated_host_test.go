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

type mockedEC2DedicatedHosts struct {
	EC2DedicatedHostsAPI
	DescribeHostsOutput ec2.DescribeHostsOutput
	ReleaseHostsOutput  ec2.ReleaseHostsOutput
}

func (m mockedEC2DedicatedHosts) DescribeHosts(ctx context.Context, params *ec2.DescribeHostsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeHostsOutput, error) {
	return &m.DescribeHostsOutput, nil
}

func (m mockedEC2DedicatedHosts) ReleaseHosts(ctx context.Context, params *ec2.ReleaseHostsInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseHostsOutput, error) {
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
			DescribeHostsOutput: ec2.DescribeHostsOutput{
				Hosts: []types.Host{
					{
						HostId: aws.String(testId1),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
						},
						AllocationTime: aws.Time(now),
					},
					{
						HostId: aws.String(testId2),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
						},
						AllocationTime: aws.Time(now.Add(1)),
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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

	err := h.nukeAll([]*string{aws.String("test-host-id-1"), aws.String("test-host-id-2")})
	require.NoError(t, err)
}
