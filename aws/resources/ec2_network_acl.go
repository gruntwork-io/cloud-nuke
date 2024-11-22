package resources

import (
	"context"
	"time"

	awsgo "github.com/aws/aws-sdk-go-v2/aws"
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
				Name:   awsgo.String("default"),
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
			DryRun:       awsgo.Bool(true),
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
	resp, err := nacl.Client.DescribeNetworkAcls(nacl.Context, &ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []string{id},
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to describe network ACL: %s", err)
		return err
	}

	if len(resp.NetworkAcls) == 0 {
		logging.Debugf("[Network ACL] Nothing found: %s", id)
		return nil
	}

	var (
		networkAcl = resp.NetworkAcls[0]
		vpcID      = networkAcl.VpcId
	)

	// Get the default nacl association id
	networkACLs, err := nacl.Client.DescribeNetworkAcls(
		nacl.Context,
		&ec2.DescribeNetworkAclsInput{
			Filters: []types.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []string{*vpcID},
				}, {
					Name:   awsgo.String("default"),
					Values: []string{"true"},
				},
			},
		},
	)
	if err != nil {
		logging.Debugf("[Network ACL] Failed to describe network ACL: %s", err)
		return err
	}

	if len(networkACLs.NetworkAcls) == 0 {
		logging.Debugf("[Network ACL] Nothing found to check the default association: %s", id)
		return nil
	}

	defaultNetworkAclID := networkACLs.NetworkAcls[0].NetworkAclId

	var associations []*types.NetworkAclAssociation
	for i := range networkAcl.Associations {
		associations = append(associations, &networkAcl.Associations[i])
	}

	err = replaceNetworkAclAssociation(nacl.Client, defaultNetworkAclID, associations)
	if err != nil {
		logging.Debugf("Failed to replace network ACL associations: %s", *defaultNetworkAclID)
		return errors.WithStackTrace(err)
	}
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
			Identifier:   awsgo.ToString(id),
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
	logging.Debugf("Start replacing network ACL associations: %s", *networkAclId)

	for _, association := range associations {
		logging.Debugf("Found %d network ACL associations to replace", len(associations))

		_, err := client.ReplaceNetworkAclAssociation(context.TODO(), &ec2.ReplaceNetworkAclAssociationInput{
			AssociationId: association.NetworkAclAssociationId,
			NetworkAclId:  networkAclId,
		})
		if err != nil {
			logging.Debugf("Failed to replace network ACL association: %s to default", *association.NetworkAclAssociationId)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Successfully replaced network ACL association: %s to default",
			*association.NetworkAclAssociationId)
	}
	logging.Debugf("Successfully replaced network ACL associations: %s", *networkAclId)
	return nil
}

func nukeNetworkAcl(client NetworkACLAPI, id *string) error {
	logging.Debugf("Deleting network Acl %s", *id)

	if _, err := client.DeleteNetworkAcl(context.TODO(), &ec2.DeleteNetworkAclInput{
		NetworkAclId: id,
	}); err != nil {
		logging.Debugf("An error happened while nuking NACL %s, error %v", *id, err)
		return err
	}
	logging.Debugf("[Ok] network acl deleted successfully %s", *id)

	return nil
}
