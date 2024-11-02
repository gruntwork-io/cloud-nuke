package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedElbV2 struct {
	LoadBalancersV2API
	DescribeLoadBalancersOutput    elasticloadbalancingv2.DescribeLoadBalancersOutput
	ErrDescribeLoadBalancersOutput error
	DeleteLoadBalancerOutput       elasticloadbalancingv2.DeleteLoadBalancerOutput
}

func (m mockedElbV2) DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	return &m.DescribeLoadBalancersOutput, m.ErrDescribeLoadBalancersOutput
}

func (m mockedElbV2) DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteLoadBalancerOutput, error) {
	return &m.DeleteLoadBalancerOutput, nil
}

func TestElbV2_GetAll(t *testing.T) {
	t.Parallel()
	testName1 := "test-name-1"
	testArn1 := "test-arn-1"
	testName2 := "test-name-2"
	testArn2 := "test-arn-2"
	now := time.Now()
	balancer := LoadBalancersV2{
		Client: mockedElbV2{
			DescribeLoadBalancersOutput: elasticloadbalancingv2.DescribeLoadBalancersOutput{
				LoadBalancers: []types.LoadBalancer{
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

type errMockLoadBalancerNotFound struct{}

func (e errMockLoadBalancerNotFound) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode(), e.ErrorMessage())
}

func (e errMockLoadBalancerNotFound) ErrorCode() string {
	return "LoadBalancerNotFound"
}

func (e errMockLoadBalancerNotFound) ErrorMessage() string {
	return "The specified load balancer does not exist."
}

func (e errMockLoadBalancerNotFound) ErrorFault() smithy.ErrorFault {
	return smithy.FaultServer
}

func TestElbV2_NukeAll(t *testing.T) {
	t.Parallel()
	var eLBNotFound errMockLoadBalancerNotFound
	balancer := LoadBalancersV2{
		BaseAwsResource: BaseAwsResource{
			Context: context.Background(),
		},
		Client: mockedElbV2{
			DescribeLoadBalancersOutput:    elasticloadbalancingv2.DescribeLoadBalancersOutput{},
			ErrDescribeLoadBalancersOutput: eLBNotFound,
			DeleteLoadBalancerOutput:       elasticloadbalancingv2.DeleteLoadBalancerOutput{},
		},
	}

	err := balancer.nukeAll([]*string{aws.String("test-arn-1")})
	require.NoError(t, err)
}
