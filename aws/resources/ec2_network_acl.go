package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeNetworkACL(networkAcl *types.NetworkAcl, firstSeenTime *time.Time, configObj config.Config) bool {
	var naclName string
	tagMap := util.ConvertTypesTagsToMap(networkAcl.Tags)
	if name, ok := tagMap["Name"]; ok {
		naclName = name
	}
	return configObj.NetworkACL.ShouldInclude(config.ResourceValue{
		Name: &naclName,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

func (nacl *NetworkACL) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var identifiers []*string

	hasMorePages := true
	params := &ec2.DescribeNetworkAclsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("default"),
				Values: []string{"false"},
			},
		},
	}

	for hasMorePages {
		resp, err := nacl.Client.DescribeNetworkAcls(ctx, params)
		if err != nil {
			logging.Debugf("[Network ACL] Failed to list network ACLs: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, networkAcl := range resp.NetworkAcls {
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, nacl.Client, networkAcl.NetworkAclId, util.ConvertTypesTagsToMap(networkAcl.Tags))
			if err != nil {
				logging.Errorf("[Network ACL] Unable to retrieve first seen tag for ACL ID: %s, error: %s", *networkAcl.NetworkAclId, err)
				continue
			}

			if shouldIncludeNetworkACL(&networkAcl, firstSeenTime, cnfObj) {
				identifiers = append(identifiers, networkAcl.NetworkAclId)
			}
		}

		params.NextToken = resp.NextToken
		hasMorePages = params.NextToken != nil
	}

	// Verify permissions for nuking NACLs
	nacl.VerifyNukablePermissions(identifiers, func(id *string) error {
		_, err := nacl.Client.DeleteNetworkAcl(ctx, &ec2.DeleteNetworkAclInput{
			NetworkAclId: id,
			DryRun:       aws.Bool(true),
		})

		return err
	})

	return identifiers, nil
}

func (nacl *NetworkACL) nuke(id *string) error {

	// nuke attached subnets
	if err := nacl.nukeAssociatedSubnets(*id); err != nil {
		return errors.WithStackTrace(err)
	}

	// nuking the network ACL
	if err := nukeNetworkAcl(nacl.Client, id); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// nukeAssociatedSubnets dissociates the specified network ACL from associated subnets and replaces the associations with the default network ACL in the same VPC.
// Important : You can't delete the ACL if it's associated with any subnets. You can't delete the default network ACL.
//
// Thus, to remove the association, it requires another network ACL ID. Here, we check the default network ACL of the VPC to which the current network ACL is attached,
// and then associate that network ACL with the association.
func (nacl *NetworkACL) nukeAssociatedSubnets(id string) error {
	logging.Debugf("[nukeAssociatedSubnets] Start describing network ACL: %s", id)

	resp, err := nacl.Client.DescribeNetworkAcls(nacl.Context, &ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []string{id},
	})
	if err != nil {
		logging.Debugf("[nukeAssociatedSubnets] Failed to describe network ACL: %s", err)
		return err
	}

	if len(resp.NetworkAcls) == 0 {
		logging.Debugf("[nukeAssociatedSubnets] No network ACL found for ID: %s", id)
		return nil
	}

	var (
		networkAcl = resp.NetworkAcls[0]
		vpcID      = networkAcl.VpcId
	)

	// Get the default network ACL association ID
	logging.Debugf("[nukeAssociatedSubnets] Describing default network ACL for VPC: %s", *vpcID)
	networkACLs, err := nacl.Client.DescribeNetworkAcls(
		nacl.Context,
		&ec2.DescribeNetworkAclsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{*vpcID},
				},
				{
					Name:   aws.String("default"),
					Values: []string{"true"},
				},
			},
		},
	)
	if err != nil {
		logging.Debugf("[nukeAssociatedSubnets] Failed to describe default network ACL: %s", err)
		return err
	}

	if len(networkACLs.NetworkAcls) == 0 {
		logging.Debugf("[nukeAssociatedSubnets] No default network ACL found for VPC: %s", *vpcID)
		return nil
	}

	defaultNetworkAclID := networkACLs.NetworkAcls[0].NetworkAclId
	logging.Debugf("[nukeAssociatedSubnets] Default network ACL ID: %s", *defaultNetworkAclID)

	var associations []*types.NetworkAclAssociation
	for i := range networkAcl.Associations {
		associations = append(associations, &networkAcl.Associations[i])
	}

	// Replacing network ACL associations
	logging.Debugf("[nukeAssociatedSubnets] Replacing network ACL associations for ID: %s", *defaultNetworkAclID)
	err = replaceNetworkAclAssociation(nacl.Client, defaultNetworkAclID, associations)
	if err != nil {
		logging.Debugf("[nukeAssociatedSubnets] Failed to replace network ACL associations: %s", *defaultNetworkAclID)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[nukeAssociatedSubnets] Successfully replaced network ACL associations for ID: %s", *defaultNetworkAclID)
	return nil
}

func (nacl *NetworkACL) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No network acl identifiers to nuke in region %s", nacl.Region)
		return nil
	}

	logging.Debugf("Deleting all network interface in region %s", nacl.Region)
	var deleted []*string

	for _, id := range identifiers {
		if nukable, reason := nacl.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}

		err := nacl.nuke(id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: "Network ACL",
			Error:        err,
		}
		report.Record(e)

		if err == nil {
			deleted = append(deleted, id)
		}
	}

	logging.Debugf("[OK] %d network interface(s) deleted in %s", len(deleted), nacl.Region)

	return nil
}

func replaceNetworkAclAssociation(client NetworkACLAPI, networkAclId *string, associations []*types.NetworkAclAssociation) error {
	logging.Debugf("[replaceNetworkAclAssociation] Start replacing network ACL associations: %s", *networkAclId)

	for _, association := range associations {
		logging.Debugf("[replaceNetworkAclAssociation] Found %d network ACL associations to replace", len(associations))

		_, err := client.ReplaceNetworkAclAssociation(context.TODO(), &ec2.ReplaceNetworkAclAssociationInput{
			AssociationId: association.NetworkAclAssociationId,
			NetworkAclId:  networkAclId,
		})
		if err != nil {
			logging.Debugf("[replaceNetworkAclAssociation] Failed to replace network ACL association: %s", *association.NetworkAclAssociationId)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("[replaceNetworkAclAssociation] Successfully replaced network ACL association: %s", *association.NetworkAclAssociationId)
	}
	logging.Debugf("[replaceNetworkAclAssociation] Successfully replaced network ACL associations: %s", *networkAclId)
	return nil
}

func nukeNetworkAcl(client NetworkACLAPI, id *string) error {
	logging.Debugf("[nukeNetworkAcl] Deleting network ACL %s", *id)

	if _, err := client.DeleteNetworkAcl(context.TODO(), &ec2.DeleteNetworkAclInput{
		NetworkAclId: id,
	}); err != nil {
		logging.Debugf("[nukeNetworkAcl] An error occurred while deleting network ACL %s: %v", *id, err)
		return err
	}
	logging.Debugf("[nukeNetworkAcl] Successfully deleted network ACL %s", *id)

	return nil
}
