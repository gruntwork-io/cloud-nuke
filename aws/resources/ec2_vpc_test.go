package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEC2VPCs struct {
	EC2VPCAPI
	DescribeVpcsOutput                             ec2.DescribeVpcsOutput
	DeleteVpcOutput                                ec2.DeleteVpcOutput
	DescribeVpcPeeringConnectionsOutput            ec2.DescribeVpcPeeringConnectionsOutput
	DescribeInstancesOutput                        ec2.DescribeInstancesOutput
	TerminateInstancesOutput                       ec2.TerminateInstancesOutput
	DescribeVpcEndpointServiceConfigurationsOutput ec2.DescribeVpcEndpointServiceConfigurationsOutput
	DeleteVpcEndpointServiceConfigurationsOutput   ec2.DeleteVpcEndpointServiceConfigurationsOutput
	DescribeInstancesError                         error
}

func (m mockedEC2VPCs) DescribeVpcs(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m mockedEC2VPCs) DeleteVpc(ctx context.Context, input *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error) {
	return &m.DeleteVpcOutput, nil
}

func (m mockedEC2VPCs) DescribeVpcPeeringConnections(ctx context.Context, input *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	return &m.DescribeVpcPeeringConnectionsOutput, nil
}

func (m mockedEC2VPCs) DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, m.DescribeInstancesError
}

func (m mockedEC2VPCs) TerminateInstances(ctx context.Context, input *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func (m mockedEC2VPCs) DescribeVpcEndpointServiceConfigurations(ctx context.Context, input *ec2.DescribeVpcEndpointServiceConfigurationsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointServiceConfigurationsOutput, error) {
	return &m.DescribeVpcEndpointServiceConfigurationsOutput, nil
}

func (m mockedEC2VPCs) DeleteVpcEndpointServiceConfigurations(ctx context.Context, input *ec2.DeleteVpcEndpointServiceConfigurationsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointServiceConfigurationsOutput, error) {
	return &m.DeleteVpcEndpointServiceConfigurationsOutput, nil
}

func TestEC2VPC_Exclude_tag(t *testing.T) {

	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true)

	testName1 := "test-vpc-name1"
	testName2 := "test-vpc-name2"
	testId1 := "test-vpc-id1"
	testId2 := "test-vpc-id2"
	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeVpcsOutput: ec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{
					{
						VpcId: awsgo.String(testId1),
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
							{
								Key:   awsgo.String("cloud-nuke-excluded"),
								Value: awsgo.String("true"),
							},
						},
					},
					{
						VpcId: awsgo.String(testId2),
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
						},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.EC2ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.EC2ResourceType{},
			expected:  []string{testId2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := vpc.getAll(tc.ctx, config.Config{
				VPC: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.ToStringSlice(names))
		})
	}

}

func TestEC2VPC_GetAll(t *testing.T) {

	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testName1 := "test-vpc-name1"
	testName2 := "test-vpc-name2"
	now := time.Now()
	testId1 := "test-vpc-id1"
	testId2 := "test-vpc-id2"
	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeVpcsOutput: ec2.DescribeVpcsOutput{
				Vpcs: []types.Vpc{
					{
						VpcId: awsgo.String(testId1),
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						VpcId: awsgo.String(testId2),
						Tags: []types.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1))),
							},
						},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.EC2ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.EC2ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.EC2ResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile(testName1),
						}},
					},
				},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.EC2ResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
					}},
			},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := vpc.getAll(tc.ctx, config.Config{
				VPC: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.ToStringSlice(names))
		})
	}
}

// TODO: Fix this test
//func TestEC2VPC_NukeAll(t *testing.T) {
//
//	t.Parallel()
//
//	vpc := EC2VPCs{
//		Client: mockedEC2VPCs{
//			DeleteVpcOutput: ec2.DeleteVpcOutput{},
//		},
//	}
//
//	err := vpc.nukeAll([]string{"test-vpc-id1", "test-vpc-id2"})
//	require.NoError(t, err)
//}

func TestEC2VPCPeeringConnections_NukeAll(t *testing.T) {
	t.Parallel()
	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeVpcPeeringConnectionsOutput: ec2.DescribeVpcPeeringConnectionsOutput{},
		},
	}

	err := nukePeeringConnections(vpc.Client, "vpc-test-00001")
	require.NoError(t, err)
}

type mockedEC2ELB struct {
	ELBClientAPI
	DescribeLoadBalancersOutput elbv2.DescribeLoadBalancersOutput
	DeleteLoadBalancerOutput    elbv2.DeleteLoadBalancerOutput

	DescribeTargetGroupsOutput elbv2.DescribeTargetGroupsOutput
	DeleteTargetGroupOutput    elbv2.DeleteTargetGroupOutput
}

func (m mockedEC2ELB) DescribeLoadBalancers(ctx context.Context, input *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
	return &m.DescribeLoadBalancersOutput, nil
}

func (m mockedEC2ELB) DescribeTargetGroups(ctx context.Context, input *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	return &m.DescribeTargetGroupsOutput, nil
}

func (m mockedEC2ELB) DeleteLoadBalancer(ctx context.Context, input *elbv2.DeleteLoadBalancerInput, optFns ...func(*elbv2.Options)) (*elbv2.DeleteLoadBalancerOutput, error) {
	return &m.DeleteLoadBalancerOutput, nil
}

func (m mockedEC2ELB) DeleteTargetGroup(ctx context.Context, input *elbv2.DeleteTargetGroupInput, optFns ...func(*elbv2.Options)) (*elbv2.DeleteTargetGroupOutput, error) {
	return &m.DeleteTargetGroupOutput, nil
}

func TestAttachedLB_Nuke(t *testing.T) {
	t.Parallel()
	vpcID := "vpc-0e9a3e7c72d9f3a0f"

	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeVpcEndpointServiceConfigurationsOutput: ec2.DescribeVpcEndpointServiceConfigurationsOutput{
				ServiceConfigurations: []types.ServiceConfiguration{
					{
						ServiceId:               aws.String("load-balancer-arn-service-id-00012"),
						GatewayLoadBalancerArns: []string{"load-balancer-arn-00012"},
					},
				},
			},
		},
		ELBClient: mockedEC2ELB{
			DescribeLoadBalancersOutput: elbv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{
					{
						VpcId:           awsgo.String(vpcID),
						LoadBalancerArn: awsgo.String("load-balancer-arn-00012"),
					},
				},
			},
		},
	}

	vpc.Context = context.Background()
	err := nukeAttachedLB(vpc.Client, vpc.ELBClient, vpcID)
	require.NoError(t, err)
}

func TestTargetGroup_Nuke(t *testing.T) {
	t.Parallel()
	vpcID := "vpc-0e9a3e7c72d9f3a0f"

	vpc := EC2VPCs{
		ELBClient: mockedEC2ELB{
			DescribeTargetGroupsOutput: elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []elbtypes.TargetGroup{
					{
						VpcId:          awsgo.String(vpcID),
						TargetGroupArn: awsgo.String("arn:aws:elasticloadbalancing:us-east-1:tg-001"),
					},
				},
			},
		},
	}

	err := nukeTargetGroups(vpc.ELBClient, vpcID)
	require.NoError(t, err)
}
