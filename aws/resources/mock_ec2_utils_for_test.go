package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func getDefaultDescribeVpcsInput() *ec2.DescribeVpcsInput {
	return &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("isDefault"),
				Values: []*string{awsgo.String("true")},
			},
		},
	}
}

func getDescribeInternetGatewaysInput(vpcId string) *ec2.DescribeInternetGatewaysInput {
	return &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("attachment.vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeInternetGatewaysOutput(gatewayId string) *ec2.DescribeInternetGatewaysOutput {
	return &ec2.DescribeInternetGatewaysOutput{
		InternetGateways: []*ec2.InternetGateway{
			{InternetGatewayId: awsgo.String(gatewayId)},
		},
	}
}

func getDetachInternetGatewayInput(vpcId, gatewayId string) *ec2.DetachInternetGatewayInput {
	return &ec2.DetachInternetGatewayInput{
		InternetGatewayId: awsgo.String(gatewayId),
		VpcId:             awsgo.String(vpcId),
	}
}

func getDeleteInternetGatewayInput(gatewayId string) *ec2.DeleteInternetGatewayInput {
	return &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: awsgo.String(gatewayId),
	}
}

func getDescribeEgressOnlyInternetGatewaysInput() *ec2.DescribeEgressOnlyInternetGatewaysInput {
	return &ec2.DescribeEgressOnlyInternetGatewaysInput{}
}

func getDescribeNetworkInterfacesInput(vpcId string) *ec2.DescribeNetworkInterfacesInput {
	return &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: awsgo.StringSlice([]string{vpcId}),
			},
		},
	}
}

func getAssociateDhcpOptionsInput(vpcId string) *ec2.AssociateDhcpOptionsInput {
	return &ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: awsgo.String("default"),
		VpcId:         awsgo.String(vpcId),
	}
}

func getDescribeSubnetsInput(vpcId string) *ec2.DescribeSubnetsInput {
	return &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeSubnetsOutput(subnetIds []string) *ec2.DescribeSubnetsOutput {
	var subnets []*ec2.Subnet
	for _, subnetId := range subnetIds {
		subnets = append(subnets, &ec2.Subnet{SubnetId: awsgo.String(subnetId)})
	}
	return &ec2.DescribeSubnetsOutput{Subnets: subnets}
}

func getDeleteSubnetInput(subnetId string) *ec2.DeleteSubnetInput {
	return &ec2.DeleteSubnetInput{
		SubnetId: awsgo.String(subnetId),
	}
}

func getDescribeRouteTablesInput(vpcId string) *ec2.DescribeRouteTablesInput {
	return &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeRouteTablesOutput(routeTableIds []string) *ec2.DescribeRouteTablesOutput {
	var routeTables []*ec2.RouteTable
	for _, routeTableId := range routeTableIds {
		routeTables = append(routeTables, &ec2.RouteTable{RouteTableId: awsgo.String(routeTableId)})
	}
	return &ec2.DescribeRouteTablesOutput{RouteTables: routeTables}
}

func getDeleteRouteTableInput(routeTableId string) *ec2.DeleteRouteTableInput {
	return &ec2.DeleteRouteTableInput{
		RouteTableId: awsgo.String(routeTableId),
	}
}

func getDescribeNetworkAclsInput(vpcId string) *ec2.DescribeNetworkAclsInput {
	return &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("default"),
				Values: []*string{awsgo.String("false")},
			},
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeNetworkAclsOutput(networkAclIds []string) *ec2.DescribeNetworkAclsOutput {
	var networkAcls []*ec2.NetworkAcl
	for _, networkAclId := range networkAclIds {
		networkAcls = append(networkAcls, &ec2.NetworkAcl{NetworkAclId: awsgo.String(networkAclId)})
	}
	return &ec2.DescribeNetworkAclsOutput{NetworkAcls: networkAcls}
}

func getDeleteNetworkAclInput(networkAclId string) *ec2.DeleteNetworkAclInput {
	return &ec2.DeleteNetworkAclInput{
		NetworkAclId: awsgo.String(networkAclId),
	}
}

func getDescribeSecurityGroupRulesInput(securityGroupId string) *ec2.DescribeSecurityGroupRulesInput {
	return &ec2.DescribeSecurityGroupRulesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("group-id"),
				Values: []*string{awsgo.String(securityGroupId)},
			},
		},
	}
}

func getDescribeSecurityGroupRulesOutput(securityGroupRuleIds []string) *ec2.DescribeSecurityGroupRulesOutput {
	var securityGroupRules []*ec2.SecurityGroupRule
	for _, securityGroupRule := range securityGroupRuleIds {
		securityGroupRules = append(securityGroupRules, &ec2.SecurityGroupRule{
			IsEgress:            awsgo.Bool(true), // egress rule
			SecurityGroupRuleId: awsgo.String(securityGroupRule),
		})
		securityGroupRules = append(securityGroupRules, &ec2.SecurityGroupRule{
			IsEgress:            awsgo.Bool(false), // ingress rule
			SecurityGroupRuleId: awsgo.String(securityGroupRule),
		})
	}
	return &ec2.DescribeSecurityGroupRulesOutput{SecurityGroupRules: securityGroupRules}
}

func getRevokeSecurityGroupEgressInput(securityGroupId string, securityGroupRuleId string) *ec2.RevokeSecurityGroupEgressInput {
	return &ec2.RevokeSecurityGroupEgressInput{
		GroupId:              awsgo.String(securityGroupId),
		SecurityGroupRuleIds: []*string{awsgo.String(securityGroupRuleId)},
	}
}

func getRevokeSecurityGroupIngressInput(securityGroupId string, securityGroupRuleId string) *ec2.RevokeSecurityGroupIngressInput {
	return &ec2.RevokeSecurityGroupIngressInput{
		GroupId:              awsgo.String(securityGroupId),
		SecurityGroupRuleIds: []*string{awsgo.String(securityGroupRuleId)},
	}
}

func getDescribeSecurityGroupsInput(vpcId string) *ec2.DescribeSecurityGroupsInput {
	return &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeSecurityGroupsOutput(securityGroupIds []string) *ec2.DescribeSecurityGroupsOutput {
	var securityGroups []*ec2.SecurityGroup
	for _, securityGroup := range securityGroupIds {
		securityGroups = append(securityGroups, &ec2.SecurityGroup{
			GroupId:   awsgo.String(securityGroup),
			GroupName: awsgo.String(""),
		})
	}
	return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: securityGroups}
}

func getDeleteSecurityGroupInput(securityGroupId string) *ec2.DeleteSecurityGroupInput {
	return &ec2.DeleteSecurityGroupInput{
		GroupId: awsgo.String(securityGroupId),
	}
}

func getDeleteVpcInput(vpcId string) *ec2.DeleteVpcInput {
	return &ec2.DeleteVpcInput{
		VpcId: awsgo.String(vpcId),
	}
}

func getDescribeSecurityGroupsInputEmpty() *ec2.DescribeSecurityGroupsInput {
	return &ec2.DescribeSecurityGroupsInput{}
}

func getDescribeDefaultSecurityGroupsOutput(groups []DefaultSecurityGroup) *ec2.DescribeSecurityGroupsOutput {
	var securityGroups []*ec2.SecurityGroup
	for _, group := range groups {
		securityGroups = append(securityGroups, &ec2.SecurityGroup{
			GroupId:   awsgo.String(group.GroupId),
			GroupName: awsgo.String("default"),
		})
	}
	return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: securityGroups}
}

func getDescribeEndpointsInput(vpcId string) *ec2.DescribeVpcEndpointsInput {
	return &ec2.DescribeVpcEndpointsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
		},
	}
}

func getDescribeEndpointsWaitForDeletionInput(vpcId string) *ec2.DescribeVpcEndpointsInput {
	return &ec2.DescribeVpcEndpointsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcId)},
			},
			{
				Name:   awsgo.String("vpc-endpoint-state"),
				Values: []*string{awsgo.String("deleting")},
			},
		},
	}
}

func getDescribeEndpointsOutput(endpointIds []string) *ec2.DescribeVpcEndpointsOutput {
	var endpoints []*ec2.VpcEndpoint
	for _, endpointId := range endpointIds {
		endpoints = append(endpoints, &ec2.VpcEndpoint{
			VpcEndpointId: &endpointId,
		})
	}

	return &ec2.DescribeVpcEndpointsOutput{
		VpcEndpoints: endpoints,
	}
}

func getDeleteEndpointInput(endpointId string) *ec2.DeleteVpcEndpointsInput {
	return &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []*string{&endpointId},
	}
}
