package resources

import (
	"context"
	cerrors "errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"

	"github.com/gruntwork-io/go-commons/retry"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (v *EC2VPCs) setFirstSeenTag(vpc ec2.Vpc, value time.Time) error {
	// We set a first seen tag because an Elastic IP doesn't contain an attribute that gives us it's creation time
	_, err := v.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{vpc.VpcId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestamp(value)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (v *EC2VPCs) getFirstSeenTag(vpc ec2.Vpc) (*time.Time, error) {
	tags := vpc.Tags
	for _, tag := range tags {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return nil, nil
}

func (v *EC2VPCs) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	// Note: This filter initially handles non-default resources and can be overridden by passing the only-default filter to choose default VPCs.
	result, err := v.Client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("is-default"),
				Values: []*string{
					awsgo.String(strconv.FormatBool(configObj.VPC.DefaultOnly)), // convert the bool status into string
				},
			},
		},
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, vpc := range result.Vpcs {
		firstSeenTime, err := v.getFirstSeenTag(*vpc)
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := v.setFirstSeenTag(*vpc, time.Now().UTC()); err != nil {
				return nil, err
			}
		}

		if configObj.VPC.ShouldInclude(config.ResourceValue{
			Time: firstSeenTime,
			Name: util.GetEC2ResourceNameTagValue(vpc.Tags),
		}) {
			ids = append(ids, vpc.VpcId)
		}
	}

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

func nuke(client ec2iface.EC2API, elbClient elbv2iface.ELBV2API, vpcID string) error {
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
			interfaces, err := client.DescribeNetworkInterfaces(
				&ec2.DescribeNetworkInterfacesInput{
					Filters: []*ec2.Filter{
						{
							Name:   awsgo.String("vpc-id"),
							Values: []*string{awsgo.String(vpcID)},
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

func nukePeeringConnections(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Finding VPC peering connections to nuke for: %s", vpcID))

	peerConnections := []*string{}
	vpcIds := []string{vpcID}
	requesterFilters := []*ec2.Filter{
		{
			Name:   awsgo.String("requester-vpc-info.vpc-id"),
			Values: awsgo.StringSlice(vpcIds),
		}, {
			Name:   awsgo.String("status-code"),
			Values: awsgo.StringSlice([]string{"active"}),
		},
	}
	accepterFilters := []*ec2.Filter{
		{
			Name:   awsgo.String("accepter-vpc-info.vpc-id"),
			Values: awsgo.StringSlice(vpcIds),
		}, {
			Name:   awsgo.String("status-code"),
			Values: awsgo.StringSlice([]string{"active"}),
		},
	}

	// check the peering connection as requester
	err := client.DescribeVpcPeeringConnectionsPages(
		&ec2.DescribeVpcPeeringConnectionsInput{
			Filters: requesterFilters,
		},
		func(page *ec2.DescribeVpcPeeringConnectionsOutput, lastPage bool) bool {
			for _, connection := range page.VpcPeeringConnections {
				peerConnections = append(peerConnections, connection.VpcPeeringConnectionId)
			}
			return !lastPage
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe vpc peering connections for vpc as requester: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	// check the peering connection as accepter
	err = client.DescribeVpcPeeringConnectionsPages(
		&ec2.DescribeVpcPeeringConnectionsInput{
			Filters: accepterFilters,
		},
		func(page *ec2.DescribeVpcPeeringConnectionsOutput, lastPage bool) bool {
			for _, connection := range page.VpcPeeringConnections {
				peerConnections = append(peerConnections, connection.VpcPeeringConnectionId)
			}
			return !lastPage
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe vpc peering connections for vpc as accepter: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Found %d VPC Peering connections to Nuke.", len(peerConnections)))

	for _, connection := range peerConnections {
		_, err := client.DeleteVpcPeeringConnection(&ec2.DeleteVpcPeeringConnectionInput{
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
func nukeInternetGateways(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking Internet Gateway for vpc: %s", vpcID))
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("attachment.vpc-id"),
				Values: []*string{awsgo.String(vpcID)},
			},
		},
	}
	igw, err := client.DescribeInternetGateways(input)
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
			awsgo.StringValue(igw.InternetGateways[0].InternetGatewayId)))
		return errors.WithStackTrace(err)
	}

	return nil
}

func nukeEgressOnlyGateways(client ec2iface.EC2API, vpcID string) error {
	var allEgressGateways []*string
	logging.Debug(fmt.Sprintf("Start nuking Egress Only Internet Gateways for vpc: %s", vpcID))
	err := client.DescribeEgressOnlyInternetGatewaysPages(
		&ec2.DescribeEgressOnlyInternetGatewaysInput{},
		func(page *ec2.DescribeEgressOnlyInternetGatewaysOutput, lastPage bool) bool {
			for _, gateway := range page.EgressOnlyInternetGateways {
				for _, attachment := range gateway.Attachments {
					if *attachment.VpcId == vpcID {
						allEgressGateways = append(allEgressGateways, gateway.EgressOnlyInternetGatewayId)
						break
					}
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe Egress Only Internet Gateways for vpc: %s", vpcID))
		return err
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

func nukeEndpoints(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking VPC endpoints for vpc: %s", vpcID))
	endpoints, _ := client.DescribeVpcEndpoints(
		&ec2.DescribeVpcEndpointsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(vpcID)},
				},
			},
		},
	)

	logging.Debug(fmt.Sprintf("Found %d VPC endpoint.", len(endpoints.VpcEndpoints)))
	var endpointIds []*string
	for _, endpoint := range endpoints.VpcEndpoints {
		// Note: sometime the state is all lower cased, sometime it is not.
		if strings.ToLower(*endpoint.State) != strings.ToLower(ec2.StateAvailable) {
			logging.Debug(fmt.Sprintf("Skipping VPC endpoint %s as it is not in available state: %s",
				awsgo.StringValue(endpoint.VpcEndpointId), awsgo.StringValue(endpoint.State)))
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
		logging.Debug(fmt.Sprintf("Failed to delete VPC endpoints: %s", err.Error()))
		return errors.WithStackTrace(err)
	}

	if err := waitForVPCEndpointToBeDeleted(client, vpcID); err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted VPC endpoints"))
	return nil
}

func nukeNetworkInterfaces(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking network interfaces for vpc: %s", vpcID))

	var allNetworkInterfaces []*ec2.NetworkInterface
	vpcIds := []string{vpcID}
	filters := []*ec2.Filter{{Name: awsgo.String("vpc-id"), Values: awsgo.StringSlice(vpcIds)}}
	err := client.DescribeNetworkInterfacesPages(
		&ec2.DescribeNetworkInterfacesInput{
			Filters: filters,
		},
		func(page *ec2.DescribeNetworkInterfacesOutput, lastPage bool) bool {
			for _, netInterface := range page.NetworkInterfaces {
				allNetworkInterfaces = append(allNetworkInterfaces, netInterface)
			}
			return !lastPage
		},
	)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe network interfaces for vpc: %s", vpcID))
		return err
	}

	logging.Debug(fmt.Sprintf("Found %d Elastic Network Interfaces to Nuke.", len(allNetworkInterfaces)))
	for _, netInterface := range allNetworkInterfaces {
		if strings.ToLower(*netInterface.Status) == strings.ToLower(ec2.NetworkInterfaceStatusInUse) {
			logging.Debug(fmt.Sprintf("Skipping network interface %s as it is in use",
				*netInterface.NetworkInterfaceId))
			continue
		}

		err = nukeNetworkInterface(client, netInterface.NetworkInterfaceId)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete network interface: %s", *netInterface))
			return errors.WithStackTrace(err)
		}

		logging.Debug(fmt.Sprintf("Successfully deleted network interface: %s", *netInterface.NetworkInterfaceId))
	}

	return nil
}

func nukeSubnets(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking subnets for vpc: %s", vpcID))
	subnets, _ := client.DescribeSubnets(
		&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(vpcID)},
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
			logging.Debug(fmt.Sprintf("Failed to delete subnet %s for vpc %s", awsgo.StringValue(subnet.SubnetId), vpcID))
			return errors.WithStackTrace(err)
		}
	}
	logging.Debug(fmt.Sprintf("Successfully deleted subnets for vpc %v", vpcID))
	return nil

}

func nukeNatGateways(client ec2iface.EC2API, vpcID string) error {
	gateways, err := client.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{awsgo.String(vpcID)},
			},
		},
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe NAT gateways for vpc: %s", vpcID))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Found %d NAT gateways to delete", len(gateways.NatGateways)))
	for _, gateway := range gateways.NatGateways {
		if *gateway.State != ec2.NatGatewayStateAvailable {
			logging.Debug(fmt.Sprintf("Skipping NAT gateway %s as it is not in available state: %s",
				awsgo.StringValue(gateway.NatGatewayId), awsgo.StringValue(gateway.State)))
			continue
		}

		err := nukeNATGateway(client, gateway.NatGatewayId)
		if err != nil {
			logging.Debug(
				fmt.Sprintf("Failed to delete NAT gateway %s for vpc %v", awsgo.StringValue(gateway.NatGatewayId), vpcID))
			return errors.WithStackTrace(err)
		}
	}
	logging.Debugf("Successfully deleted NAT gateways for vpc %s", vpcID)

	return nil
}

func nukeRouteTables(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking route tables for vpc: %s", vpcID))
	routeTables, err := client.DescribeRouteTables(
		&ec2.DescribeRouteTablesInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(vpcID)},
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
				awsgo.StringValue(routeTable.RouteTableId)))
			continue
		}

		logging.Debug(fmt.Sprintf("Start nuking route table: %s", awsgo.StringValue(routeTable.RouteTableId)))
		for _, association := range routeTable.Associations {
			_, err := client.DisassociateRouteTable(&ec2.DisassociateRouteTableInput{
				AssociationId: association.RouteTableAssociationId,
			})
			if err != nil {
				logging.Debug(fmt.Sprintf("Failed to disassociate route table: %s",
					awsgo.StringValue(association.RouteTableAssociationId)))
				return errors.WithStackTrace(err)
			}
			logging.Debug(fmt.Sprintf("Successfully disassociated route table: %s",
				awsgo.StringValue(association.RouteTableAssociationId)))
		}

		_, err := client.DeleteRouteTable(
			&ec2.DeleteRouteTableInput{
				RouteTableId: routeTable.RouteTableId,
			},
		)
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete route table: %s", awsgo.StringValue(routeTable.RouteTableId)))
			return errors.WithStackTrace(err)
		}
		logging.Debug(fmt.Sprintf("Successfully deleted route table: %s", awsgo.StringValue(routeTable.RouteTableId)))
	}

	return nil
}

// nukeNacls nukes all network ACLs in a VPC except the default one. It replaces all subnet associations with
// the default in order to prevent dependency violations.
// You can't delete the ACL if it's associated with any subnets.
// You can't delete the default network ACL.
//
// https://docs.aws.amazon.com/cli/latest/reference/ec2/delete-network-acl.html
func nukeNacls(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking network ACLs for vpc: %s", vpcID))
	networkACLs, _ := client.DescribeNetworkAcls(
		&ec2.DescribeNetworkAclsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(vpcID)},
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

		logging.Debugf("Start nuking network ACL: %s", awsgo.StringValue(networkACL.NetworkAclId))
		err := replaceNetworkAclAssociation(client, defaultNetworkAclID, networkACL.Associations)
		if err != nil {
			logging.Debugf("Failed to replace network ACL associations: %s", awsgo.StringValue(networkACL.NetworkAclId))
			return errors.WithStackTrace(err)
		}

		err = nukeNetworkAcl(client, networkACL.NetworkAclId)
		if err != nil {
			logging.Debugf("Failed to delete network ACL: %s", awsgo.StringValue(networkACL.NetworkAclId))
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Successfully deleted network ACL: %s", awsgo.StringValue(networkACL.NetworkAclId))
	}

	return nil
}

func nukeSecurityGroups(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Start nuking security groups for vpc: %s", vpcID))
	securityGroups, _ := client.DescribeSecurityGroups(
		&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{awsgo.String(vpcID)},
				},
			},
		},
	)

	logging.Debug(fmt.Sprintf("Found %d security groups to delete", len(securityGroups.SecurityGroups)))
	for _, securityGroup := range securityGroups.SecurityGroups {
		securityGroupRules, _ := client.DescribeSecurityGroupRules(
			&ec2.DescribeSecurityGroupRulesInput{
				Filters: []*ec2.Filter{
					{
						Name:   awsgo.String("group-id"),
						Values: []*string{securityGroup.GroupId},
					},
				},
			},
		)

		logging.Debug(fmt.Sprintf("Found %d security rules to delete for security group: %s",
			len(securityGroupRules.SecurityGroupRules), awsgo.StringValue(securityGroup.GroupId)))
		for _, securityGroupRule := range securityGroupRules.SecurityGroupRules {
			logging.Debug(fmt.Sprintf("Deleting Security Group Rule %s", awsgo.StringValue(securityGroupRule.SecurityGroupRuleId)))
			if *securityGroupRule.IsEgress {
				_, err := client.RevokeSecurityGroupEgress(
					&ec2.RevokeSecurityGroupEgressInput{
						GroupId:              securityGroup.GroupId,
						SecurityGroupRuleIds: []*string{securityGroupRule.SecurityGroupRuleId},
					},
				)
				if err != nil {
					logging.Debug(fmt.Sprintf("Failed to revoke security group egress rule: %s",
						awsgo.StringValue(securityGroupRule.SecurityGroupRuleId)))
					return errors.WithStackTrace(err)
				}
				logging.Debug(fmt.Sprintf("Successfully revoked security group egress rule: %s",
					awsgo.StringValue(securityGroupRule.SecurityGroupRuleId)))
			} else {
				_, err := client.RevokeSecurityGroupIngress(
					&ec2.RevokeSecurityGroupIngressInput{
						GroupId:              securityGroup.GroupId,
						SecurityGroupRuleIds: []*string{securityGroupRule.SecurityGroupRuleId},
					},
				)
				if err != nil {
					logging.Debug(fmt.Sprintf("Failed to revoke security group ingress rule: %s",
						awsgo.StringValue(securityGroupRule.SecurityGroupRuleId)))
					return errors.WithStackTrace(err)
				}
				logging.Debug(fmt.Sprintf("Successfully revoked security group ingress rule: %s",
					awsgo.StringValue(securityGroupRule.SecurityGroupRuleId)))
			}
		}
	}

	for _, securityGroup := range securityGroups.SecurityGroups {
		if *securityGroup.GroupName != "default" {
			logging.Debug(fmt.Sprintf("Deleting Security Group %s for vpc %s", awsgo.StringValue(securityGroup.GroupId), vpcID))

			err := nukeSecurityGroup(client, securityGroup.GroupId)
			if err != nil {
				logging.Debug(fmt.Sprintf(
					"Successfully deleted security group %s", awsgo.StringValue(securityGroup.GroupId)))
				return errors.WithStackTrace(err)
			}
			logging.Debug(fmt.Sprintf(
				"Successfully deleted security group %s", awsgo.StringValue(securityGroup.GroupId)))
		}
	}

	return nil
}

// default option is not available for this, and it only supports deleting non-default resources
// https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-dhcp-options.html
func nukeDhcpOptions(client ec2iface.EC2API, vpcID string) error {

	// Deletes the specified set of DHCP options. You must disassociate the set of DHCP options before you can delete it.
	// You can disassociate the set of DHCP options by associating either a new set of options or the default set of
	// options with the VPC.
	vpcs, err := client.DescribeVpcs(
		&ec2.DescribeVpcsInput{
			VpcIds: []*string{awsgo.String(vpcID)},
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
	_, err = client.AssociateDhcpOptions(&ec2.AssociateDhcpOptionsInput{
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

func nukeVpc(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Deleting VPC %s", vpcID))
	input := &ec2.DeleteVpcInput{
		VpcId: awsgo.String(vpcID),
	}
	_, err := client.DeleteVpc(input)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete VPC %s", vpcID))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted VPC %s", vpcID))
	return nil
}

func nukeAttachedLB(client ec2iface.EC2API, elbclient elbv2iface.ELBV2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Describing load balancers for %s", vpcID))

	// get all loadbalancers
	output, err := elbclient.DescribeLoadBalancers(nil)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe loadbalancer for %s", vpcID))
		return errors.WithStackTrace(err)
	}
	var attachedLoadBalancers []string

	// get a list of load balancers which was attached on the vpc
	for _, lb := range output.LoadBalancers {
		if awsgo.StringValue(lb.VpcId) != vpcID {
			continue
		}

		attachedLoadBalancers = append(attachedLoadBalancers, awsgo.StringValue(lb.LoadBalancerArn))
	}

	// check the load-balancers are attached with any vpc-endpoint-service, then nuke them first
	esoutput, err := client.DescribeVpcEndpointServiceConfigurations(nil)
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe vpc endpoint services for %s", vpcID))
		return errors.WithStackTrace(err)
	}

	// since we don't want duplicating the endpoint service ids, using a map here
	var nukableEndpointServices = make(map[*string]struct{})
	for _, config := range esoutput.ServiceConfigurations {
		// check through gateway load balancer attachments and select the service for nuking
		for _, gwlb := range config.GatewayLoadBalancerArns {
			if slices.Contains(attachedLoadBalancers, awsgo.StringValue(gwlb)) {
				nukableEndpointServices[config.ServiceId] = struct{}{}
			}
		}

		// check through network load balancer attachments and select the service for nuking
		for _, nwlb := range config.NetworkLoadBalancerArns {
			if slices.Contains(attachedLoadBalancers, awsgo.StringValue(nwlb)) {
				nukableEndpointServices[config.ServiceId] = struct{}{}
			}
		}
	}

	logging.Debug(fmt.Sprintf("Found %d Endpoint services attached with the load balancers to nuke.", len(nukableEndpointServices)))

	// nuke the endpoint services
	for endpointService := range nukableEndpointServices {
		_, err := client.DeleteVpcEndpointServiceConfigurations(&ec2.DeleteVpcEndpointServiceConfigurationsInput{
			ServiceIds: []*string{endpointService},
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete endpoint service %v for %s", awsgo.StringValue(endpointService), vpcID))
			return errors.WithStackTrace(err)
		}
	}
	logging.Debug(fmt.Sprintf("Successfully deleted he endpoints attached with load balancers for %s", vpcID))

	// nuke the load-balancers
	for _, lb := range attachedLoadBalancers {
		_, err := elbclient.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{
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

func nukeTargetGroups(client elbv2iface.ELBV2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Describing target groups for %s", vpcID))

	output, err := client.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to describe target groups for %s", vpcID))
		return errors.WithStackTrace(err)
	}

	for _, tg := range output.TargetGroups {
		// if the target group is not for this vpc, then skip
		if tg.VpcId != nil && awsgo.StringValue(tg.VpcId) == vpcID {
			_, err := client.DeleteTargetGroup(&elbv2.DeleteTargetGroupInput{
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

func nukeEc2Instances(client ec2iface.EC2API, vpcID string) error {
	logging.Debug(fmt.Sprintf("Describing instances for %s", vpcID))
	output, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("network-interface.vpc-id"),
				Values: awsgo.StringSlice([]string{
					vpcID,
				}),
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

		_, err := client.TerminateInstances(&ec2.TerminateInstancesInput{
			InstanceIds: terminateInstances,
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to terminate instances for %s", vpcID))
			return errors.WithStackTrace(err)
		}

		// weight for terminate the instances
		logging.Debug(fmt.Sprintf("waiting for the instance to be terminated for %s", vpcID))
		err = client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
			InstanceIds: terminateInstances,
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to wait instance termination for %s", vpcID))
			return errors.WithStackTrace(err)
		}
	}

	logging.Debug(fmt.Sprintf("Successfully deleted instances for %s", vpcID))
	return nil
}
