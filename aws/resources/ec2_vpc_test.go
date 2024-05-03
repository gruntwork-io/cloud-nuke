package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEC2VPCs struct {
	ec2iface.EC2API
	DescribeVpcsOutput                             ec2.DescribeVpcsOutput
	DeleteVpcOutput                                ec2.DeleteVpcOutput
	DescribeVpcPeeringConnectionsOutput            ec2.DescribeVpcPeeringConnectionsOutput
	DescribeInstancesOutput                        ec2.DescribeInstancesOutput
	TerminateInstancesOutput                       ec2.TerminateInstancesOutput
	DescribeVpcEndpointServiceConfigurationsOutput ec2.DescribeVpcEndpointServiceConfigurationsOutput
	DeleteVpcEndpointServiceConfigurationsOutput   ec2.DeleteVpcEndpointServiceConfigurationsOutput
}

func (m mockedEC2VPCs) DescribeVpcs(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m mockedEC2VPCs) DeleteVpc(input *ec2.DeleteVpcInput) (*ec2.DeleteVpcOutput, error) {
	return &m.DeleteVpcOutput, nil
}
func (m mockedEC2VPCs) DescribeVpcPeeringConnectionsPages(input *ec2.DescribeVpcPeeringConnectionsInput, callback func(page *ec2.DescribeVpcPeeringConnectionsOutput, lastPage bool) bool) error {
	callback(&m.DescribeVpcPeeringConnectionsOutput, true)
	return nil
}
func (m mockedEC2VPCs) DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, nil
}

func (m mockedEC2VPCs) TerminateInstances(*ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}
func (m mockedEC2VPCs) WaitUntilInstanceTerminated(*ec2.DescribeInstancesInput) error {
	return nil
}

func (m mockedEC2VPCs) DescribeVpcEndpointServiceConfigurations(*ec2.DescribeVpcEndpointServiceConfigurationsInput) (*ec2.DescribeVpcEndpointServiceConfigurationsOutput, error) {
	return &m.DescribeVpcEndpointServiceConfigurationsOutput, nil
}
func (m mockedEC2VPCs) DeleteVpcEndpointServiceConfigurations(*ec2.DeleteVpcEndpointServiceConfigurationsInput) (*ec2.DeleteVpcEndpointServiceConfigurationsOutput, error) {
	return &m.DeleteVpcEndpointServiceConfigurationsOutput, nil
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
				Vpcs: []*ec2.Vpc{
					{
						VpcId: awsgo.String(testId1),
						Tags: []*ec2.Tag{
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
						Tags: []*ec2.Tag{
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
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
	elbv2iface.ELBV2API
	DescribeLoadBalancersOutput elbv2.DescribeLoadBalancersOutput
	DeleteLoadBalancerOutput    elbv2.DeleteLoadBalancerOutput

	DescribeTargetGroupsOutput elbv2.DescribeTargetGroupsOutput
	DeleteTargetGroupOutput    elbv2.DeleteTargetGroupOutput
}

func (m mockedEC2ELB) DescribeLoadBalancers(*elbv2.DescribeLoadBalancersInput) (*elbv2.DescribeLoadBalancersOutput, error) {
	return &m.DescribeLoadBalancersOutput, nil
}
func (m mockedEC2ELB) DescribeTargetGroups(*elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
	return &m.DescribeTargetGroupsOutput, nil
}

func (m mockedEC2ELB) DeleteLoadBalancer(*elbv2.DeleteLoadBalancerInput) (*elbv2.DeleteLoadBalancerOutput, error) {
	return &m.DeleteLoadBalancerOutput, nil
}
func (m mockedEC2ELB) DeleteTargetGroup(*elbv2.DeleteTargetGroupInput) (*elbv2.DeleteTargetGroupOutput, error) {
	return &m.DeleteTargetGroupOutput, nil
}

func TestAttachedLB_Nuke(t *testing.T) {
	t.Parallel()
	vpcID := "vpc-0e9a3e7c72d9f3a0f"

	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeVpcEndpointServiceConfigurationsOutput: ec2.DescribeVpcEndpointServiceConfigurationsOutput{
				ServiceConfigurations: []*ec2.ServiceConfiguration{
					{
						GatewayLoadBalancerArns: awsgo.StringSlice([]string{
							"load-balancer-arn-00012",
						}),
					},
				},
			},
		},
		ELBClient: mockedEC2ELB{
			DescribeLoadBalancersOutput: elbv2.DescribeLoadBalancersOutput{
				LoadBalancers: []*elbv2.LoadBalancer{
					{
						VpcId:           awsgo.String(vpcID),
						LoadBalancerArn: awsgo.String("load-balancer-arn-00012"),
					},
				},
			},
		},
	}

	err := nukeAttachedLB(vpc.Client, vpc.ELBClient, vpcID)
	require.NoError(t, err)
}

func TestTargetGroup_Nuke(t *testing.T) {
	t.Parallel()
	vpcID := "vpc-0e9a3e7c72d9f3a0f"

	vpc := EC2VPCs{
		ELBClient: mockedEC2ELB{
			DescribeTargetGroupsOutput: elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []*elbv2.TargetGroup{
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

func TestEc2Instance_Nuke(t *testing.T) {
	t.Parallel()
	vpcID := "vpc-0e9a3e7c72d9f3a0f"

	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeInstancesOutput: ec2.DescribeInstancesOutput{
				Reservations: []*ec2.Reservation{
					{
						Instances: []*ec2.Instance{
							{
								InstanceId: awsgo.String("instance-001"),
							},
						},
					},
				},
			},
		},
	}

	err := nukeEc2Instances(vpc.Client, vpcID)
	require.NoError(t, err)
}
