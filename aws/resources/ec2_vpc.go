package resources

import (
	"context"
	cerrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

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

func (v *EC2VPCs) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := v.Client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			// Note: this filter omits the default since there is special
			// handling for default resources already
			{
				Name:   awsgo.String("is-default"),
				Values: awsgo.StringSlice([]string{"false"}),
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
		err = nuke(v.Client, id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   id,
			ResourceType: "VPC",
			Error:        err,
		}
		report.Record(e)

		if err != nil {

			pterm.Error.Println(fmt.Sprintf("Failed to nuke vpc with err: %s", err))
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking VPC",
			}, map[string]interface{}{
				"region": v.Region,
			})
			multierror.Append(multiErr, err)
		} else {
			deletedVPCs++
			logging.Debug(fmt.Sprintf("Deleted VPC: %s", id))
		}
	}

	return nil
}

func nuke(client ec2iface.EC2API, vpcID string) error {
	// Note: order is quite important, otherwise you will encounter dependency violation errors.
	logging.Debug(fmt.Sprintf("Start nuking VPC %s", vpcID))
	err := nukeDhcpOptions(client, vpcID)
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

	err = nukeNatGateway(client, vpcID)
	if err != nil {
		logging.Debug(fmt.Sprintf("Error cleaning up NAT Gateways for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeEgressOnlyGateways(client, vpcID)
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Error cleaning up Egress Only Internet Gateways for VPC %s: %s", vpcID, err.Error()))
		return err
	}

	err = nukeInternetGateway(client, vpcID)
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

	err = nukeVpc(client, vpcID)
	if err != nil {
		pterm.Error.Println(fmt.Sprintf("Error deleting VPC %s: %s ", vpcID, err))
		return err
	}

	logging.Debug(fmt.Sprintf("Successfully nuked VPC %s", vpcID))
	logging.Debug("")
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
	var allErrs *multierror.Error
	for _, gateway := range allEgressGateways {
		_, err := client.DeleteEgressOnlyInternetGateway(
			&ec2.DeleteEgressOnlyInternetGatewayInput{EgressOnlyInternetGatewayId: gateway})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to delete Egress Only Internet Gateway %s", *gateway))
			allErrs = multierror.Append(allErrs, errors.WithStackTrace(err))
		}

		logging.Debug(fmt.Sprintf("Successfully deleted Egress Only Internet Gateway %s", *gateway))
	}

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

	_, err := client.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: endpointIds,
	})
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

func waitForVPCEndpointToBeDeleted(client ec2iface.EC2API, vpcID string) error {
	for i := 0; i < 30; i++ {
		endpoints, err := client.DescribeVpcEndpoints(
			&ec2.DescribeVpcEndpointsInput{
				Filters: []*ec2.Filter{
					{
						Name:   awsgo.String("vpc-id"),
						Values: []*string{awsgo.String(vpcID)},
					},
					{
						Name:   awsgo.String("vpc-endpoint-state"),
						Values: []*string{awsgo.String("deleting")},
					},
				},
			},
		)
		if err != nil {
			return err
		}

		if len(endpoints.VpcEndpoints) == 0 {
			return nil
		}

		time.Sleep(20 * time.Second)
		logging.Debug(fmt.Sprintf("Waiting for VPC endpoints to be deleted..."))
	}

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

		_, err = client.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: netInterface.NetworkInterfaceId,
		})
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

	logging.Debug(fmt.Sprintf("Found %d subnets to delete ", len(subnets.Subnets)))
	if len(subnets.Subnets) > 0 {
		for _, subnet := range subnets.Subnets {
			_, err := client.DeleteSubnet(
				&ec2.DeleteSubnetInput{
					SubnetId: subnet.SubnetId,
				},
			)

			if err != nil {
				logging.Debug(fmt.Sprintf("Failed to delete subnet %s", awsgo.StringValue(subnet.SubnetId)))
				return errors.WithStackTrace(err)
			}
			logging.Debug(fmt.Sprintf("Successfully deleted subnet %s", awsgo.StringValue(subnet.SubnetId)))
		}

		return nil
	}

	logging.Debug(fmt.Sprintf("No subnets found"))
	return nil
}

func nukeNatGateway(client ec2iface.EC2API, vpcID string) error {
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

		_, err := client.DeleteNatGateway(&ec2.DeleteNatGatewayInput{
			NatGatewayId: gateway.NatGatewayId,
		})
		if err != nil {
			logging.Debug(
				fmt.Sprintf("Failed to delete NAT gateway %s", awsgo.StringValue(gateway.NatGatewayId)))
			return errors.WithStackTrace(err)
		}
		logging.Debug(
			fmt.Sprintf("Successfully deleted NAT gateway %s", awsgo.StringValue(gateway.NatGatewayId)))
	}

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

		logging.Debug(fmt.Sprintf(
			"Start nuking network ACL: %s", awsgo.StringValue(networkACL.NetworkAclId)))
		for _, association := range networkACL.Associations {
			logging.Debug(fmt.Sprintf(
				"Found %d network ACL associations to replace", len(networkACL.Associations)))
			_, err := client.ReplaceNetworkAclAssociation(&ec2.ReplaceNetworkAclAssociationInput{
				AssociationId: association.NetworkAclAssociationId,
				NetworkAclId:  defaultNetworkAclID,
			})
			if err != nil {
				logging.Debug(fmt.Sprintf("Failed to replace network ACL association: %s to default",
					awsgo.StringValue(association.NetworkAclAssociationId)))
				return errors.WithStackTrace(err)
			}
			logging.Debug(fmt.Sprintf("Successfully replaced network ACL association: %s to default",
				awsgo.StringValue(association.NetworkAclAssociationId)))
		}

		_, err := client.DeleteNetworkAcl(
			&ec2.DeleteNetworkAclInput{
				NetworkAclId: networkACL.NetworkAclId,
			},
		)
		if err != nil {
			logging.Debug(fmt.Sprintf(
				"Failed to delete network ACL: %s", awsgo.StringValue(networkACL.NetworkAclId)))
			return errors.WithStackTrace(err)
		}
		logging.Debug(fmt.Sprintf(
			"Successfully deleted network ACL: %s", awsgo.StringValue(networkACL.NetworkAclId)))
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
			_, err := client.DeleteSecurityGroup(
				&ec2.DeleteSecurityGroupInput{
					GroupId: securityGroup.GroupId,
				},
			)
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
