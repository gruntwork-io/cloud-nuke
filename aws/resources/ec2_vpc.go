package resources

import (
	"context"
	cerrors "errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"

	"github.com/gruntwork-io/go-commons/retry"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (v *EC2VPCs) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var firstSeenTime *time.Time
	var err error
	// Note: This filter initially handles non-default resources and can be overridden by passing the only-default filter to choose default VPCs.
	result, err := v.Client.DescribeVpcs(v.Context, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   awsgo.String("is-default"),
				Values: []string{strconv.FormatBool(configObj.VPC.DefaultOnly)}, // convert the bool status into string
			},
		},
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, vpc := range result.Vpcs {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, v.Client, vpc.VpcId, util.ConvertTypesTagsToMap(vpc.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if configObj.VPC.ShouldInclude(config.ResourceValue{
			Time: firstSeenTime,
			Name: util.GetEC2ResourceNameTagValue(vpc.Tags),
		}) {
			ids = append(ids, vpc.VpcId)
		}
	}

	// checking the nukable permissions
	v.VerifyNukablePermissions(ids, func(id *string) error {
		_, err := v.Client.DeleteVpc(c, &ec2.DeleteVpcInput{
			VpcId:  id,
			DryRun: awsgo.Bool(true),
		})
		return err
	})

	return ids, nil
}

func (v *EC2VPCs) nukeAll(vpcIds []string) error {
	if len(vpcIds) == 0 {
		logging.Debug("No VPCs to nuke")
		return nil
	}

	logging.Debug("Deleting all VPCs")

	deletedVPCs := 0
	multiErr := new(multierror.Error)

	for _, id := range vpcIds {
		if nukable, reason := v.IsNukable(id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", id, reason)
			continue
		}

		var err error
		err = nuke(v.Client, v.ELBClient, id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   id,
			ResourceType: "VPC",
			Error:        err,
		}
		report.Record(e)

		if err != nil {

			pterm.Error.Println(fmt.Sprintf("Failed to nuke vpc with err: %s", err))
			multierror.Append(multiErr, err)
		} else {
			deletedVPCs++
			logging.Debug(fmt.Sprintf("Deleted VPC: %s", id))
		}
	}

	return nil
}

func nuke(client EC2VPCAPI, elbClient ELBClientAPI, vpcID string) error {
	var err error
	// Note: order is quite important, otherwise you will encounter dependency violation errors.

	err = nukeAttachedLB(client, elbClient, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error nuking loadbalancer for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeTargetGroups(elbClient, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error nuking target group for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeEc2Instances(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error nuking instances for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	logging.Debug(fmt.Sprintf("Start nuking VPC %s", vpcID))
	err = nukeDhcpOptions(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up DHCP Options for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeRouteTables(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up Route Tables for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeNetworkInterfaces(client, vpcID)
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Error cleaning up Elastic Network Interfaces for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeNatGateways(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up NAT Gateways for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeEgressOnlyGateways(client, vpcID)
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Error cleaning up Egress Only Internet Gateways for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	// NOTE: Since the network interfaces attached to the load balancer may not be removed immediately after removing the load balancers,
	// and attempting to remove the internet gateway without waiting for these network interfaces removal will result in an error.
	// The actual error message states: 'has some mapped public address(es). Please unmap those public address(es) before detaching the gateway.'
	// Therefore, it is recommended to wait until all the load balancer-related network interfaces are detached and deleted before proceeding.
	//
	// Important : The waiting should be happen before nuking the internet gateway
	err = retry.DoWithRetry(
		logging.Logger.WithTime(time.Now()),
		"Waiting for all Network interfaces to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			interfaces, err := client.DescribeNetworkInterfaces(context.Background(),
				&ec2.DescribeNetworkInterfacesInput{
					Filters: []types.Filter{
						{
							Name:   awsgo.String("vpc-id"),
							Values: []string{vpcID},
						},
					},
				},
			)
			if err != nil {
				return errors.WithStackTrace(err)
			}

			if len(interfaces.NetworkInterfaces) == 0 {
				return nil
			}

			return fmt.Errorf("Not all Network interfaces are deleted.")
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error waiting up Network interfaces deletion for VPC %s: %s", vpcID, err.Error()))
		return errors.WithStackTrace(err)
	}

	err = nukeInternetGateways(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up Internet Gateway for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeNacls(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up Network ACLs for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeSubnets(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up Subnets for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeEndpoints(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up Endpoints for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeSecurityGroups(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up Security Groups for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukePeeringConnections(client, vpcID)
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Error deleting VPC Peer connection %s: %s ", vpcID, err))
		return err
	}

	err = nukeVpc(client, vpcID)
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Error deleting VPC %s: %s ", vpcID, err))
		return err
	}

	logging.Debug(fmt.Sprintf("Successfully nuked VPC %s", vpcID))
	logging.Debug("")
	return nil
}

func nukePeeringConnections(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Finding VPC peering connections to nuke for: %s", vpcID))

	peerConnections := []*string{}
	vpcIds := []string{vpcID}
	requesterFilters := []types.Filter{
		{
			Name:   awsgo.String("requester-vpc-info.vpc-id"),
			Values: vpcIds,
		}, {
			Name:   awsgo.String("status-code"),
			Values: []string{"active"},
		},
	}
	accepterFilters := []types.Filter{
		{
			Name:   awsgo.String("accepter-vpc-info.vpc-id"),
			Values: vpcIds,
		}, {
			Name:   awsgo.String("status-code"),
			Values: []string{"active"},
		},
	}

	// check the peering connection as requester
	paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{
		Filters: requesterFilters,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to describe VPC peering connections for VPC as requester: %s", vpcID))
			return errors.WithStackTrace(err)
		}
		for _, connection := range page.VpcPeeringConnections {
			peerConnections = append(peerConnections, connection.VpcPeeringConnectionId)
		}
	}

	// check the peering connection as accepter
	paginator = ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{
		Filters: accepterFilters,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to describe VPC peering connections for VPC as accepter: %s", vpcID))
			return errors.WithStackTrace(err)
		}
		for _, connection := range page.VpcPeeringConnections {
			peerConnections = append(peerConnections, connection.VpcPeeringConnectionId)
		}
	}

	logging.Debug(fmt.Sprintf("Found %d VPC Peering connections to Nuke.", len(peerConnections)))

	for _, connection := range peerConnections {
		_, err := client.DeleteVpcPeeringConnection(context.Background(), &ec2.DeleteVpcPeeringConnectionInput{
			VpcPeeringConnectionId: connection,
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete peering connection for vpc: %s", vpcID))
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

// nukeVpcInternetGateways
// This function is specifically for VPCs. It retrieves all the internet gateways attached to the given VPC ID and nuke them
func nukeInternetGateways(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking Internet Gateway for vpc: %s", vpcID))
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   awsgo.String("attachment.vpc-id"),
				Values: []string{vpcID},
			},
		},
	}
	igw, err := client.DescribeInternetGateways(context.Background(), input)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe internet gateways for vpc: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	if len(igw.InternetGateways) < 1 {
		logging.Debug(fmt.Sprintf("No Internet Gateway to delete."))
		return nil
	}

	// re-using the method inside internet gateway
	err = nukeInternetGateway(client, igw.InternetGateways[0].InternetGatewayId, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete internet gateway %s",
			awsgo.ToString(igw.InternetGateways[0].InternetGatewayId)))
		return errors.WithStackTrace(err)
	}

	return nil
}

func nukeEgressOnlyGateways(client EC2VPCAPI, vpcID string) error {
	var allEgressGateways []*string
	logging.Debug(fmt.Sprintf("Start nuking Egress Only Internet Gateways for vpc: %s", vpcID))
	paginator := ec2.NewDescribeEgressOnlyInternetGatewaysPaginator(client, &ec2.DescribeEgressOnlyInternetGatewaysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to describe Egress Only Internet Gateways for vpc: %s", vpcID))
			return err
		}
		for _, gateway := range page.EgressOnlyInternetGateways {
			for _, attachment := range gateway.Attachments {
				if *attachment.VpcId == vpcID {
					allEgressGateways = append(allEgressGateways, gateway.EgressOnlyInternetGatewayId)
					break
				}
			}
		}
	}

	logging.Debug(fmt.Sprintf("Found %d Egress Only Internet Gateways to nuke.", len(allEgressGateways)))

	for _, gateway := range allEgressGateways {
		err := nukeEgressOnlyGateway(client, gateway)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete Egress Only Internet Gateway %s for vpc %s", *gateway, vpcID))
			return errors.WithStackTrace(err)
		}
	}

	logging.Debug(fmt.Sprintf("Successfully deleted Egress Only Internet Gateways for %s", vpcID))

	return nil
}

func nukeEndpoints(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking VPC endpoints for vpc: %s", vpcID))
	endpoints, _ := client.DescribeVpcEndpoints(context.Background(),
		&ec2.DescribeVpcEndpointsInput{
			Filters: []types.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []string{vpcID},
				},
			},
		},
	)

	logging.Debug(fmt.Sprintf("Found %d VPC endpoint.", len(endpoints.VpcEndpoints)))
	var endpointIds []*string
	for _, endpoint := range endpoints.VpcEndpoints {
		// Note: sometime the state is all lower cased, sometime it is not.
		if strings.ToLower(string(endpoint.State)) != strings.ToLower(string(types.StateAvailable)) {
			logging.Debug(fmt.Sprintf("Skipping VPC endpoint %s as it is not in available state: %s",
				awsgo.ToString(endpoint.VpcEndpointId), endpoint.State))
			continue
		}

		endpointIds = append(endpointIds, endpoint.VpcEndpointId)
	}

	if len(endpointIds) == 0 {
		logging.Debug(fmt.Sprintf("No VPC endpoint to nuke."))
		return nil
	}
	err := nukeVpcEndpoint(client, endpointIds)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete VPC endpoints: %v", err))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted VPC endpoints"))
	return nil
}

func nukeNetworkInterfaces(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking network interfaces for vpc: %s", vpcID))

	var allNetworkInterfaces []types.NetworkInterface
	vpcIds := []string{vpcID}
	filters := []types.Filter{{Name: awsgo.String("vpc-id"), Values: vpcIds}}
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{
		Filters: filters,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to describe network interfaces for vpc: %s", vpcID))
			return err
		}
		for _, netInterface := range page.NetworkInterfaces {
			allNetworkInterfaces = append(allNetworkInterfaces, netInterface)
		}
	}

	logging.Debug(fmt.Sprintf("Found %d Elastic Network Interfaces to Nuke.", len(allNetworkInterfaces)))
	for _, netInterface := range allNetworkInterfaces {
		if strings.ToLower(string(netInterface.Status)) == strings.ToLower(string(types.NetworkInterfaceStatusInUse)) {
			logging.Debug(fmt.Sprintf("Skipping network interface %s as it is in use",
				*netInterface.NetworkInterfaceId))
			continue
		}

		err := nukeNetworkInterface(client, netInterface.NetworkInterfaceId)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete network interface: %v", netInterface))
			return errors.WithStackTrace(err)
		}

		logging.Debug(fmt.Sprintf("Successfully deleted network interface: %s", *netInterface.NetworkInterfaceId))
	}

	return nil
}

func nukeSubnets(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking subnets for vpc: %s", vpcID))
	subnets, _ := client.DescribeSubnets(context.Background(),
		&ec2.DescribeSubnetsInput{
			Filters: []types.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []string{vpcID},
				},
			},
		},
	)

	if len(subnets.Subnets) == 0 {
		logging.Debug(fmt.Sprintf("No subnets found"))
		return nil
	}

	logging.Debug(fmt.Sprintf("Found %d subnets to delete ", len(subnets.Subnets)))
	for _, subnet := range subnets.Subnets {
		err := nukeSubnet(client, subnet.SubnetId)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete subnet %s for vpc %s", awsgo.ToString(subnet.SubnetId), vpcID))
			return errors.WithStackTrace(err)
		}
	}
	logging.Debug(fmt.Sprintf("Successfully deleted subnets for vpc %v", vpcID))
	return nil

}

func nukeNatGateways(client EC2VPCAPI, vpcID string) error {
	gateways, err := client.DescribeNatGateways(context.Background(), &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   awsgo.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe NAT gateways for vpc: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Found %d NAT gateways to delete", len(gateways.NatGateways)))
	for _, gateway := range gateways.NatGateways {
		if gateway.State != types.NatGatewayStateAvailable {
			logging.Debug(fmt.Sprintf("Skipping NAT gateway %s as it is not in available state: %s",
				awsgo.ToString(gateway.NatGatewayId), gateway.State))
			continue
		}

		err := nukeNATGateway(client, gateway.NatGatewayId)
		if err != nil {
			logging.Debug(
				fmt.Sprintf("Failed to delete NAT gateway %s for vpc %v", awsgo.ToString(gateway.NatGatewayId), vpcID))
			return errors.WithStackTrace(err)
		}
	}
	logging.Debugf("Successfully deleted NAT gateways for vpc %s", vpcID)

	return nil
}

func nukeRouteTables(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking route tables for vpc: %s", vpcID))
	routeTables, err := client.DescribeRouteTables(context.Background(),
		&ec2.DescribeRouteTablesInput{
			Filters: []types.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []string{vpcID},
				},
			},
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe route tables for vpc: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Found %d route tables.", len(routeTables.RouteTables)))
	for _, routeTable := range routeTables.RouteTables {
		// Skip main route table
		if len(routeTable.Associations) > 0 && *routeTable.Associations[0].Main {
			logging.Debug(fmt.Sprintf("Skipping main route table: %s",
				awsgo.ToString(routeTable.RouteTableId)))
			continue
		}

		logging.Debug(fmt.Sprintf("Start nuking route table: %s", awsgo.ToString(routeTable.RouteTableId)))
		for _, association := range routeTable.Associations {
			_, err := client.DisassociateRouteTable(context.Background(), &ec2.DisassociateRouteTableInput{
				AssociationId: association.RouteTableAssociationId,
			})
			if err != nil {
				logging.Debug(fmt.Sprintf("Failed to disassociate route table: %s",
					awsgo.ToString(association.RouteTableAssociationId)))
				return errors.WithStackTrace(err)
			}
			logging.Debug(fmt.Sprintf("Successfully disassociated route table: %s",
				awsgo.ToString(association.RouteTableAssociationId)))
		}

		_, err := client.DeleteRouteTable(context.Background(),
			&ec2.DeleteRouteTableInput{
				RouteTableId: routeTable.RouteTableId,
			},
		)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete route table: %s", awsgo.ToString(routeTable.RouteTableId)))
			return errors.WithStackTrace(err)
		}
		logging.Debug(fmt.Sprintf("Successfully deleted route table: %s", awsgo.ToString(routeTable.RouteTableId)))
	}

	return nil
}

// nukeNacls nukes all network ACLs in a VPC except the default one. It replaces all subnet associations with
// the default in order to prevent dependency violations.
// You can't delete the ACL if it's associated with any subnets.
// You can't delete the default network ACL.
//
// https://docs.aws.amazon.com/cli/latest/reference/ec2/delete-network-acl.html
func nukeNacls(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking network ACLs for vpc: %s", vpcID))
	networkACLs, _ := client.DescribeNetworkAcls(context.Background(),
		&ec2.DescribeNetworkAclsInput{
			Filters: []types.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []string{vpcID},
				},
			},
		},
	)

	// Find the default network ACL to connect all subnets to.
	var defaultNetworkAclID *string
	for _, networkACL := range networkACLs.NetworkAcls {
		if *networkACL.IsDefault {
			defaultNetworkAclID = networkACL.NetworkAclId
			break
		}
	}

	if defaultNetworkAclID == nil {
		logging.Debug(fmt.Sprintf("No default network ACL found"))
		return cerrors.New("No default network ACL found in vpc: " + vpcID)
	}

	logging.Debug(fmt.Sprintf("Found default network ACL: %s", *defaultNetworkAclID))
	logging.Debug(fmt.Sprintf("Found %d network ACLs to delete", len(networkACLs.NetworkAcls)-1))
	for _, networkACL := range networkACLs.NetworkAcls {
		if *networkACL.IsDefault {
			continue
		}

		logging.Debugf("Start nuking network ACL: %s", awsgo.ToString(networkACL.NetworkAclId))
		associations := make([]*types.NetworkAclAssociation, len(networkACL.Associations))
		for i := range networkACL.Associations {
			associations[i] = &networkACL.Associations[i]
		}
		err := replaceNetworkAclAssociation(client, defaultNetworkAclID, associations)
		if err != nil {
			logging.Debugf("Failed to replace network ACL associations: %s", awsgo.ToString(networkACL.NetworkAclId))
			return errors.WithStackTrace(err)
		}

		err = nukeNetworkAcl(client, networkACL.NetworkAclId)
		if err != nil {
			logging.Debugf("Failed to delete network ACL: %s", awsgo.ToString(networkACL.NetworkAclId))
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Successfully deleted network ACL: %s", awsgo.ToString(networkACL.NetworkAclId))
	}

	return nil
}

func nukeSecurityGroups(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking security groups for vpc: %s", vpcID))
	securityGroups, _ := client.DescribeSecurityGroups(context.Background(),
		&ec2.DescribeSecurityGroupsInput{
			Filters: []types.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []string{vpcID},
				},
			},
		},
	)

	logging.Debug(fmt.Sprintf("Found %d security groups to delete", len(securityGroups.SecurityGroups)))
	for _, securityGroup := range securityGroups.SecurityGroups {
		securityGroupRules, _ := client.DescribeSecurityGroupRules(context.Background(),
			&ec2.DescribeSecurityGroupRulesInput{
				Filters: []types.Filter{
					{
						Name:   awsgo.String("group-id"),
						Values: []string{*securityGroup.GroupId},
					},
				},
			},
		)

		logging.Debug(fmt.Sprintf("Found %d security rules to delete for security group: %s",
			len(securityGroupRules.SecurityGroupRules), awsgo.ToString(securityGroup.GroupId)))
		for _, securityGroupRule := range securityGroupRules.SecurityGroupRules {
			logging.Debug(fmt.Sprintf("Deleting Security Group Rule %s", awsgo.ToString(securityGroupRule.SecurityGroupRuleId)))
			if *securityGroupRule.IsEgress {
				_, err := client.RevokeSecurityGroupEgress(context.Background(),
					&ec2.RevokeSecurityGroupEgressInput{
						GroupId:              securityGroup.GroupId,
						SecurityGroupRuleIds: []string{*securityGroupRule.SecurityGroupRuleId},
					},
				)
				if err != nil {
					logging.Debug(fmt.Sprintf("Failed to revoke security group egress rule: %s",
						awsgo.ToString(securityGroupRule.SecurityGroupRuleId)))
					return errors.WithStackTrace(err)
				}
				logging.Debug(fmt.Sprintf("Successfully revoked security group egress rule: %s",
					awsgo.ToString(securityGroupRule.SecurityGroupRuleId)))
			} else {
				_, err := client.RevokeSecurityGroupIngress(context.Background(),
					&ec2.RevokeSecurityGroupIngressInput{
						GroupId:              securityGroup.GroupId,
						SecurityGroupRuleIds: []string{*securityGroupRule.SecurityGroupRuleId},
					},
				)
				if err != nil {
					logging.Debug(fmt.Sprintf("Failed to revoke security group ingress rule: %s",
						awsgo.ToString(securityGroupRule.SecurityGroupRuleId)))
					return errors.WithStackTrace(err)
				}
				logging.Debug(fmt.Sprintf("Successfully revoked security group ingress rule: %s",
					awsgo.ToString(securityGroupRule.SecurityGroupRuleId)))
			}
		}
	}

	for _, securityGroup := range securityGroups.SecurityGroups {
		if *securityGroup.GroupName != "default" {
			logging.Debug(fmt.Sprintf("Deleting Security Group %s for vpc %s", awsgo.ToString(securityGroup.GroupId), vpcID))

			err := nukeSecurityGroup(client, securityGroup.GroupId)
			if err != nil {
				logging.Debug(fmt.Sprintf(
					"Successfully deleted security group %s", awsgo.ToString(securityGroup.GroupId)))
				return errors.WithStackTrace(err)
			}
			logging.Debug(fmt.Sprintf(
				"Successfully deleted security group %s", awsgo.ToString(securityGroup.GroupId)))
		}
	}

	return nil
}

// default option is not available for this, and it only supports deleting non-default resources
// https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-dhcp-options.html
func nukeDhcpOptions(client EC2VPCAPI, vpcID string) error {

	// Deletes the specified set of DHCP options. You must disassociate the set of DHCP options before you can delete it.
	// You can disassociate the set of DHCP options by associating either a new set of options or the default set of
	// options with the VPC.
	vpcs, err := client.DescribeVpcs(context.Background(),
		&ec2.DescribeVpcsInput{
			VpcIds: []string{vpcID},
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe DHCP options associated with VPC w/ err: %s.", err))
		return errors.WithStackTrace(err)
	}

	dhcpOptionID := vpcs.Vpcs[0].DhcpOptionsId
	logging.Debug(fmt.Sprintf("Successfully found DHCP option associated with VPC: %s.", *dhcpOptionID))

	if *dhcpOptionID == "default" {
		logging.Debug(fmt.Sprintf("No DHCP options to nuke. DHCP already set to default."))
		return nil
	}

	// Disassociates a set of DHCP options from a VPC by setting the options to default.
	_, err = client.AssociateDhcpOptions(context.Background(), &ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: awsgo.String("default"),
		VpcId:         awsgo.String(vpcID),
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to associate VPC with default set of DHCP options w/ err: %s.", err))
		return errors.WithStackTrace(err)
	}
	logging.Debug(fmt.Sprintf("Successfully associated VPC with default set of DHCP options."))
	return err
}

func nukeVpc(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Deleting VPC %s", vpcID))
	input := &ec2.DeleteVpcInput{
		VpcId: awsgo.String(vpcID),
	}
	_, err := client.DeleteVpc(context.Background(), input)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete VPC %s", vpcID))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted VPC %s", vpcID))
	return nil
}

func nukeAttachedLB(client EC2VPCAPI, elbclient ELBClientAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Describing load balancers for %s", vpcID))

	// get all loadbalancers
	output, err := elbclient.DescribeLoadBalancers(context.Background(), &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe loadbalancer for %s", vpcID))
		return errors.WithStackTrace(err)
	}
	var attachedLoadBalancers []string

	// get a list of load balancers which was attached on the vpc
	for _, lb := range output.LoadBalancers {
		if awsgo.ToString(lb.VpcId) != vpcID {
			continue
		}

		attachedLoadBalancers = append(attachedLoadBalancers, awsgo.ToString(lb.LoadBalancerArn))
	}

	// check the load-balancers are attached with any vpc-endpoint-service, then nuke them first
	esoutput, err := client.DescribeVpcEndpointServiceConfigurations(context.Background(), &ec2.DescribeVpcEndpointServiceConfigurationsInput{})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe vpc endpoint services for %s", vpcID))
		return errors.WithStackTrace(err)
	}

	// since we don't want duplicating the endpoint service ids, using a map here
	var nukableEndpointServices = make(map[*string]struct{})
	for _, config := range esoutput.ServiceConfigurations {
		// check through gateway load balancer attachments and select the service for nuking
		for _, gwlb := range config.GatewayLoadBalancerArns {
			if slices.Contains(attachedLoadBalancers, gwlb) {
				nukableEndpointServices[config.ServiceId] = struct{}{}
			}
		}

		// check through network load balancer attachments and select the service for nuking
		for _, nwlb := range config.NetworkLoadBalancerArns {
			if slices.Contains(attachedLoadBalancers, nwlb) {
				nukableEndpointServices[config.ServiceId] = struct{}{}
			}
		}
	}

	logging.Debug(fmt.Sprintf("Found %d Endpoint services attached with the load balancers to nuke.", len(nukableEndpointServices)))

	// nuke the endpoint services
	for endpointService := range nukableEndpointServices {
		_, err := client.DeleteVpcEndpointServiceConfigurations(context.Background(), &ec2.DeleteVpcEndpointServiceConfigurationsInput{
			ServiceIds: []string{*endpointService},
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete endpoint service %v for %s", awsgo.ToString(endpointService), vpcID))
			return errors.WithStackTrace(err)
		}
	}
	logging.Debug(fmt.Sprintf("Successfully deleted he endpoints attached with load balancers for %s", vpcID))

	// nuke the load-balancers
	for _, lb := range attachedLoadBalancers {
		_, err := elbclient.DeleteLoadBalancer(context.Background(), &elasticloadbalancingv2.DeleteLoadBalancerInput{
			LoadBalancerArn: awsgo.String(lb),
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete loadbalancer %v for %s", lb, vpcID))
			return errors.WithStackTrace(err)
		}
	}

	logging.Debug(fmt.Sprintf("Successfully deleted loadbalancers attached %s", vpcID))
	return nil
}

func nukeTargetGroups(client ELBClientAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Describing target groups for %s", vpcID))

	output, err := client.DescribeTargetGroups(context.Background(), &elasticloadbalancingv2.DescribeTargetGroupsInput{})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe target groups for %s", vpcID))
		return errors.WithStackTrace(err)
	}

	for _, tg := range output.TargetGroups {
		// if the target group is not for this vpc, then skip
		if tg.VpcId != nil && awsgo.ToString(tg.VpcId) == vpcID {
			_, err := client.DeleteTargetGroup(context.Background(), &elasticloadbalancingv2.DeleteTargetGroupInput{
				TargetGroupArn: tg.TargetGroupArn,
			})
			if err != nil {
				logging.Debug(fmt.Sprintf("Failed to delete target group %v for %s", *tg.TargetGroupArn, vpcID))
				return errors.WithStackTrace(err)
			}
		}

	}

	logging.Debug(fmt.Sprintf("Successfully deleted target group attached %s", vpcID))

	return nil
}

func nukeEc2Instances(client EC2VPCAPI, vpcID string) error {
	logging.Debug(fmt.Sprintf("Describing instances for %s", vpcID))
	output, err := client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   awsgo.String("network-interface.vpc-id"),
				Values: []string{vpcID},
			},
		},
	})

	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe instances for %s", vpcID))
		return errors.WithStackTrace(err)
	}

	var terminateInstances []*string
	for _, instance := range output.Reservations {
		for _, i := range instance.Instances {
			terminateInstances = append(terminateInstances, i.InstanceId)
		}
	}

	if len(terminateInstances) > 0 {
		logging.Debug(fmt.Sprintf("Found %d VPC attached instances to Nuke.", len(terminateInstances)))

		// Convert []*string to []string
		terminateInstancesIds := make([]string, len(terminateInstances))
		for i, id := range terminateInstances {
			terminateInstancesIds[i] = aws.ToString(id)
		}
		_, err := client.TerminateInstances(context.Background(), &ec2.TerminateInstancesInput{
			InstanceIds: terminateInstancesIds,
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to terminate instances for %s", vpcID))
			return errors.WithStackTrace(err)
		}

		// weight for terminate the instances
		logging.Debug(fmt.Sprintf("waiting for the instance to be terminated for %s", vpcID))
		waiter := ec2.NewInstanceTerminatedWaiter(client)
		err = waiter.Wait(context.Background(), &ec2.DescribeInstancesInput{
			InstanceIds: terminateInstancesIds,
		}, 5*time.Minute)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to wait instance termination for %s", vpcID))
			return errors.WithStackTrace(err)
		}
	}

	logging.Debug(fmt.Sprintf("Successfully deleted instances for %s", vpcID))
	return nil
}
