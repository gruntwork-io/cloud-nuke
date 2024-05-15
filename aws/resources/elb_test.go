package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedLoadBalancers struct {
	elbiface.ELBAPI
	DescribeLoadBalancersOutput elb.DescribeLoadBalancersOutput
	DeleteLoadBalancerOutput    elb.DeleteLoadBalancerOutput
	DescribeLoadBalancersError  error
}

func (m mockedLoadBalancers) DescribeLoadBalancersWithContext(_ aws.Context, input *elb.DescribeLoadBalancersInput, _ ...request.Option) (*elb.DescribeLoadBalancersOutput, error) {
	return &m.DescribeLoadBalancersOutput, nil
}

func (m mockedLoadBalancers) DeleteLoadBalancerWithContext(_ aws.Context, input *elb.DeleteLoadBalancerInput, _ ...request.Option) (*elb.DeleteLoadBalancerOutput, error) {
	return &m.DeleteLoadBalancerOutput, m.DescribeLoadBalancersError
}

func TestElb_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-name-1"
	testName2 := "test-name-2"
	now := time.Now()
	balancer := LoadBalancers{
		Client: mockedLoadBalancers{
			DescribeLoadBalancersOutput: elb.DescribeLoadBalancersOutput{
				LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
					{
						LoadBalancerName: aws.String(testName1),
						CreatedTime:      aws.Time(now),
					},
					{
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
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := balancer.getAll(context.Background(), config.Config{
				ELBv1: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestElb_NukeAll(t *testing.T) {

	t.Parallel()

	balancer := LoadBalancers{
		Client: mockedLoadBalancers{
			DeleteLoadBalancerOutput:   elb.DeleteLoadBalancerOutput{},
			DescribeLoadBalancersError: awserr.New("LoadBalancerNotFound", "", nil),
		},
	}

	err := balancer.nukeAll([]*string{aws.String("test-arn-1"), aws.String("test-arn-2")})
	require.NoError(t, err)
}
