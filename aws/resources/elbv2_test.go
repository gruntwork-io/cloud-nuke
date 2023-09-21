package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedElbV2 struct {
	elbv2iface.ELBV2API
	DescribeLoadBalancersOutput elbv2.DescribeLoadBalancersOutput
	DeleteLoadBalancerOutput    elbv2.DeleteLoadBalancerOutput
}

func (m mockedElbV2) DescribeLoadBalancers(input *elbv2.DescribeLoadBalancersInput) (*elbv2.DescribeLoadBalancersOutput, error) {
	return &m.DescribeLoadBalancersOutput, nil
}

func (m mockedElbV2) DeleteLoadBalancer(input *elbv2.DeleteLoadBalancerInput) (*elbv2.DeleteLoadBalancerOutput, error) {
	return &m.DeleteLoadBalancerOutput, nil
}

func (m mockedElbV2) WaitUntilLoadBalancersDeleted(input *elbv2.DescribeLoadBalancersInput) error {
	return nil
}

func TestElbV2_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-name-1"
	testArn1 := "test-arn-1"
	testName2 := "test-name-2"
	testArn2 := "test-arn-2"
	now := time.Now()
	balancer := LoadBalancersV2{
		Client: mockedElbV2{
			DescribeLoadBalancersOutput: elbv2.DescribeLoadBalancersOutput{
				LoadBalancers: []*elbv2.LoadBalancer{
					{
						LoadBalancerArn:  aws.String(testArn1),
						LoadBalancerName: aws.String(testName1),
						CreatedTime:      aws.Time(now),
					},
					{
						LoadBalancerArn:  aws.String(testArn2),
						LoadBalancerName: aws.String(testName2),
						CreatedTime:      aws.Time(now.Add(1)),
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
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testArn1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := balancer.getAll(context.Background(), config.Config{
				ELBv2: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestElbV2_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	balancer := LoadBalancersV2{
		Client: mockedElbV2{
			DeleteLoadBalancerOutput: elbv2.DeleteLoadBalancerOutput{},
		},
	}

	err := balancer.nukeAll([]*string{aws.String("test-arn-1"), aws.String("test-arn-2")})
	require.NoError(t, err)
}
