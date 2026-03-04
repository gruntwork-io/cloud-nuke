package resources

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2VpcAPI is the interface for the EC2 VPC client.
type EC2VpcAPI interface {
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	// Safety net dependencies
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DeleteRouteTable(ctx context.Context, params *ec2.DeleteRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteTableOutput, error)
	DisassociateRouteTable(ctx context.Context, params *ec2.DisassociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateRouteTableOutput, error)
	DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	RevokeSecurityGroupIngress(ctx context.Context, input *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error)
	RevokeSecurityGroupEgress(ctx context.Context, input *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error)
	DeleteSecurityGroup(ctx context.Context, input *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error)
	DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error)
	DetachNetworkInterface(ctx context.Context, params *ec2.DetachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DetachNetworkInterfaceOutput, error)
	DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
	DetachInternetGateway(ctx context.Context, params *ec2.DetachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachInternetGatewayOutput, error)
	DeleteInternetGateway(ctx context.Context, params *ec2.DeleteInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteInternetGatewayOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
	DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error)
	DeleteVpcPeeringConnection(ctx context.Context, params *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error)
	DescribeVpnGateways(ctx context.Context, params *ec2.DescribeVpnGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpnGatewaysOutput, error)
	DetachVpnGateway(ctx context.Context, params *ec2.DetachVpnGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachVpnGatewayOutput, error)
	DeleteVpnGateway(ctx context.Context, params *ec2.DeleteVpnGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpnGatewayOutput, error)
}

// NewEC2VPC creates a new EC2 VPC resource using the generic resource pattern.
func NewEC2VPC() AwsResource {
	return NewEC2AwsResource[EC2VpcAPI](
		"vpc",
		WrapAwsInitClient(func(r *resource.Resource[EC2VpcAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		func(c config.Config) config.EC2ResourceType { return c.VPC },
		listVPCs,
		resource.MultiStepDeleter(cleanupVPCDependencies, deleteVPC),
		nil,
	)
}

func listVPCs(ctx context.Context, client EC2VpcAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
	var ids []*string
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{strconv.FormatBool(defaultOnly)},
			},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, vpc := range page.Vpcs {
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, vpc.VpcId, util.ConvertTypesTagsToMap(vpc.Tags))
			if err != nil {
				logging.Errorf("Unable to retrieve first seen tag for VPC %s: %v", aws.ToString(vpc.VpcId), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: firstSeenTime,
				Name: util.GetEC2ResourceNameTagValue(vpc.Tags),
				Tags: util.ConvertTypesTagsToMap(vpc.Tags),
			}) {
				ids = append(ids, vpc.VpcId)
			}
		}
	}

	return ids, nil
}

// cleanupVPCDependencies performs best-effort cleanup of remaining VPC dependencies.
// This is a safety net — the primary resource types (route tables, security groups, etc.)
// should already be deleted by their own resource handlers. This catches anything that
// slipped through due to ordering issues or partial failures upstream.
func cleanupVPCDependencies(ctx context.Context, client EC2VpcAPI, id *string) error {
	vpcID := aws.ToString(id)
	logging.Debugf("[VPC Safety Net] Cleaning up remaining dependencies for VPC %s", vpcID)

	cleanupVPCPeeringConnections(ctx, client, vpcID)
	cleanupVPCVpnGateways(ctx, client, vpcID)
	cleanupVPCRouteTables(ctx, client, vpcID)
	cleanupVPCNetworkInterfaces(ctx, client, vpcID)
	cleanupVPCSecurityGroups(ctx, client, vpcID)
	cleanupVPCSubnets(ctx, client, vpcID)
	cleanupVPCInternetGateways(ctx, client, vpcID)

	logging.Debugf("[VPC Safety Net] Finished dependency cleanup for VPC %s", vpcID)
	return nil
}

// cleanupVPCRouteTables deletes non-main route tables in the VPC (best-effort).
func cleanupVPCRouteTables(ctx context.Context, client EC2VpcAPI, vpcID string) {
	paginator := ec2.NewDescribeRouteTablesPaginator(client, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe route tables for %s: %s", vpcID, err)
			return
		}

		for _, rt := range page.RouteTables {
			// Skip main route tables — they are deleted automatically with the VPC
			if isMainRouteTable(rt) {
				continue
			}

			rtID := aws.ToString(rt.RouteTableId)

			// Disassociate all subnet associations first
			for _, assoc := range rt.Associations {
				if assoc.Main != nil && *assoc.Main {
					continue
				}
				if _, err := client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
					AssociationId: assoc.RouteTableAssociationId,
				}); err != nil {
					logging.Debugf("[VPC Safety Net] Failed to disassociate route table %s: %s", rtID, err)
				}
			}

			if _, err := client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
				RouteTableId: rt.RouteTableId,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to delete route table %s: %s", rtID, err)
			} else {
				logging.Debugf("[VPC Safety Net] Deleted route table %s", rtID)
			}
		}
	}
}

// cleanupVPCSecurityGroups revokes cross-group rules and deletes non-default security groups (best-effort).
func cleanupVPCSecurityGroups(ctx context.Context, client EC2VpcAPI, vpcID string) {
	// Collect all security groups first so we can do two passes (revoke rules, then delete)
	var allGroups []types.SecurityGroup
	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe security groups for %s: %s", vpcID, err)
			return
		}
		allGroups = append(allGroups, page.SecurityGroups...)
	}

	// First pass: revoke all ingress/egress rules that reference other groups in this VPC
	for _, sg := range allGroups {
		sgID := aws.ToString(sg.GroupId)

		if len(sg.IpPermissions) > 0 {
			if _, err := client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissions,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to revoke ingress for %s: %s", sgID, err)
			}
		}

		if len(sg.IpPermissionsEgress) > 0 {
			if _, err := client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissionsEgress,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to revoke egress for %s: %s", sgID, err)
			}
		}
	}

	// Second pass: delete non-default security groups
	for _, sg := range allGroups {
		if aws.ToString(sg.GroupName) == "default" {
			continue
		}

		sgID := aws.ToString(sg.GroupId)
		if _, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		}); err != nil {
			logging.Debugf("[VPC Safety Net] Failed to delete security group %s: %s", sgID, err)
		} else {
			logging.Debugf("[VPC Safety Net] Deleted security group %s", sgID)
		}
	}
}

// cleanupVPCNetworkInterfaces detaches and deletes remaining ENIs in the VPC (best-effort).
// Uses a two-pass approach: detach all ENIs first, wait for them to become available, then delete.
func cleanupVPCNetworkInterfaces(ctx context.Context, client EC2VpcAPI, vpcID string) {
	var allENIs []types.NetworkInterface
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe network interfaces for %s: %s", vpcID, err)
			return
		}
		allENIs = append(allENIs, page.NetworkInterfaces...)
	}

	// First pass: detach all attached ENIs
	var detachedENIIDs []string
	for _, eni := range allENIs {
		eniID := aws.ToString(eni.NetworkInterfaceId)

		if eni.Attachment != nil && eni.Attachment.AttachmentId != nil {
			if _, err := client.DetachNetworkInterface(ctx, &ec2.DetachNetworkInterfaceInput{
				AttachmentId: eni.Attachment.AttachmentId,
				Force:        aws.Bool(true),
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to detach ENI %s: %s", eniID, err)
			} else {
				detachedENIIDs = append(detachedENIIDs, eniID)
			}
		}
	}

	// Wait for detached ENIs to reach "available" status before deleting
	if len(detachedENIIDs) > 0 {
		waiter := ec2.NewNetworkInterfaceAvailableWaiter(client)
		if err := waiter.Wait(ctx, &ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: detachedENIIDs,
		}, 2*time.Minute); err != nil {
			logging.Debugf("[VPC Safety Net] Timed out waiting for ENIs to become available in %s: %s", vpcID, err)
		}
	}

	// Second pass: delete all ENIs
	for _, eni := range allENIs {
		eniID := aws.ToString(eni.NetworkInterfaceId)
		if _, err := client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: eni.NetworkInterfaceId,
		}); err != nil {
			logging.Debugf("[VPC Safety Net] Failed to delete ENI %s: %s", eniID, err)
		} else {
			logging.Debugf("[VPC Safety Net] Deleted ENI %s", eniID)
		}
	}
}

// cleanupVPCVpnGateways detaches and deletes remaining VPN gateways (best-effort).
// Detachment is async, so we poll for completion before deleting.
func cleanupVPCVpnGateways(ctx context.Context, client EC2VpcAPI, vpcID string) {
	// DescribeVpnGateways does not support pagination — it returns all results in one call.
	output, err := client.DescribeVpnGateways(ctx, &ec2.DescribeVpnGatewaysInput{
		Filters: []types.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("state"), Values: []string{"available"}},
		},
	})
	if err != nil {
		logging.Debugf("[VPC Safety Net] Failed to describe VPN gateways for %s: %s", vpcID, err)
		return
	}

	for _, vgw := range output.VpnGateways {
		vgwID := aws.ToString(vgw.VpnGatewayId)

		if _, err := client.DetachVpnGateway(ctx, &ec2.DetachVpnGatewayInput{
			VpnGatewayId: vgw.VpnGatewayId,
			VpcId:        aws.String(vpcID),
		}); err != nil {
			logging.Debugf("[VPC Safety Net] Failed to detach VPN gateway %s: %s", vgwID, err)
		}

		// Poll until detachment completes — no AWS waiter exists for VPN gateways
		isDetached := func(ctx context.Context) (bool, error) {
			out, err := client.DescribeVpnGateways(ctx, &ec2.DescribeVpnGatewaysInput{
				VpnGatewayIds: []string{vgwID},
			})
			if err != nil {
				return false, err
			}
			if len(out.VpnGateways) == 0 {
				return true, nil
			}
			for _, att := range out.VpnGateways[0].VpcAttachments {
				if aws.ToString(att.VpcId) == vpcID && att.State != types.AttachmentStatusDetached {
					return false, nil
				}
			}
			return true, nil
		}
		if err := util.PollUntil(ctx, "VPN gateway detach "+vgwID, 5*time.Second, 2*time.Minute, isDetached); err != nil {
			logging.Debugf("[VPC Safety Net] Timed out waiting for VPN gateway %s to detach: %s", vgwID, err)
		}

		if _, err := client.DeleteVpnGateway(ctx, &ec2.DeleteVpnGatewayInput{
			VpnGatewayId: vgw.VpnGatewayId,
		}); err != nil {
			logging.Debugf("[VPC Safety Net] Failed to delete VPN gateway %s: %s", vgwID, err)
		} else {
			logging.Debugf("[VPC Safety Net] Deleted VPN gateway %s", vgwID)
		}
	}
}

// cleanupVPCInternetGateways detaches and deletes remaining internet gateways (best-effort).
func cleanupVPCInternetGateways(ctx context.Context, client EC2VpcAPI, vpcID string) {
	paginator := ec2.NewDescribeInternetGatewaysPaginator(client, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: []string{vpcID}},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe internet gateways for %s: %s", vpcID, err)
			return
		}

		for _, igw := range page.InternetGateways {
			igwID := aws.ToString(igw.InternetGatewayId)

			if _, err := client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: igw.InternetGatewayId,
				VpcId:             aws.String(vpcID),
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to detach IGW %s: %s", igwID, err)
			}

			if _, err := client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
				InternetGatewayId: igw.InternetGatewayId,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to delete IGW %s: %s", igwID, err)
			} else {
				logging.Debugf("[VPC Safety Net] Deleted IGW %s", igwID)
			}
		}
	}
}

// cleanupVPCSubnets deletes remaining subnets in the VPC (best-effort).
func cleanupVPCSubnets(ctx context.Context, client EC2VpcAPI, vpcID string) {
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe subnets for %s: %s", vpcID, err)
			return
		}

		for _, subnet := range page.Subnets {
			subnetID := aws.ToString(subnet.SubnetId)
			if _, err := client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
				SubnetId: subnet.SubnetId,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to delete subnet %s: %s", subnetID, err)
			} else {
				logging.Debugf("[VPC Safety Net] Deleted subnet %s", subnetID)
			}
		}
	}
}

// cleanupVPCPeeringConnections deletes active VPC peering connections for the VPC (best-effort).
func cleanupVPCPeeringConnections(ctx context.Context, client EC2VpcAPI, vpcID string) {
	// Check peering connections where this VPC is the requester
	requesterPaginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{
		Filters: []types.Filter{
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("status-code"), Values: []string{"active", "pending-acceptance", "provisioning"}},
		},
	})

	for requesterPaginator.HasMorePages() {
		page, err := requesterPaginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe requester peering connections for %s: %s", vpcID, err)
			break
		}

		for _, pcx := range page.VpcPeeringConnections {
			pcxID := aws.ToString(pcx.VpcPeeringConnectionId)
			if _, err := client.DeleteVpcPeeringConnection(ctx, &ec2.DeleteVpcPeeringConnectionInput{
				VpcPeeringConnectionId: pcx.VpcPeeringConnectionId,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to delete peering connection %s: %s", pcxID, err)
			} else {
				logging.Debugf("[VPC Safety Net] Deleted peering connection %s", pcxID)
			}
		}
	}

	// Check peering connections where this VPC is the accepter
	accepterPaginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{
		Filters: []types.Filter{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("status-code"), Values: []string{"active", "pending-acceptance", "provisioning"}},
		},
	})

	for accepterPaginator.HasMorePages() {
		page, err := accepterPaginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[VPC Safety Net] Failed to describe accepter peering connections for %s: %s", vpcID, err)
			break
		}

		for _, pcx := range page.VpcPeeringConnections {
			pcxID := aws.ToString(pcx.VpcPeeringConnectionId)
			if _, err := client.DeleteVpcPeeringConnection(ctx, &ec2.DeleteVpcPeeringConnectionInput{
				VpcPeeringConnectionId: pcx.VpcPeeringConnectionId,
			}); err != nil {
				logging.Debugf("[VPC Safety Net] Failed to delete peering connection %s: %s", pcxID, err)
			} else {
				logging.Debugf("[VPC Safety Net] Deleted peering connection %s", pcxID)
			}
		}
	}
}

func deleteVPC(ctx context.Context, client EC2VpcAPI, id *string) error {
	logging.Debugf("Deleting VPC %s", aws.ToString(id))

	if _, err := client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: id,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Successfully deleted VPC %s", aws.ToString(id))
	return nil
}
