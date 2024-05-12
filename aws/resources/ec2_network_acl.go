package resources

import (
	"context"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeNetworkACL(networkAcl *ec2.NetworkAcl, firstSeenTime *time.Time, configObj config.Config) bool {
	var naclName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(networkAcl.Tags)
	if name, ok := tagMap["Name"]; ok {
		naclName = name
	}
	return configObj.NetworkACL.ShouldInclude(config.ResourceValue{
		Name: &naclName,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

func (nacl *NetworkACL) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	var firstSeenTime *time.Time
	var err error

	resp, err := nacl.Client.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("default"),
				Values: []*string{
					awsgo.String("false"), // can't able to nuke default nacl
				},
			},
		},
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to list network ACL: %s", err)
		return nil, err
	}

	for _, networkAcl := range resp.NetworkAcls {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, nacl.Client, networkAcl.NetworkAclId, util.ConvertEC2TagsToMap(networkAcl.Tags))
		if err != nil {
			logging.Error("unable to retrieve first seen tag")
			continue
		}

		if shouldIncludeNetworkACL(networkAcl, firstSeenTime, configObj) {
			identifiers = append(identifiers, networkAcl.NetworkAclId)
		}
	}

	// Check and verify the list of allowed nuke actions
	nacl.VerifyNukablePermissions(identifiers, func(id *string) error {
		_, err := nacl.Client.DeleteNetworkAcl(&ec2.DeleteNetworkAclInput{
			NetworkAclId: id,
			DryRun:       awsgo.Bool(true),
		})
		return err
	})

	return identifiers, nil
}

func (nacl *NetworkACL) nuke(id *string) error {

	// nuke attached subnets
	if err := nacl.nukeAssociatedSubnets(id); err != nil {
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
func (nacl *NetworkACL) nukeAssociatedSubnets(id *string) error {
	resp, err := nacl.Client.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []*string{id},
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to describe network ACL: %s", err)
		return err
	}

	if len(resp.NetworkAcls) == 0 {
		logging.Debugf("[Network ACL] Nothing found: %s", awsgo.StringValue(id))
		return nil
	}

	var (
		networkAcl = resp.NetworkAcls[0]
		vpcID      = networkAcl.VpcId // get the vpc of this nacl
	)

	// Get the default nacl association id
	networkACLs, err := nacl.Client.DescribeNetworkAcls(
		&ec2.DescribeNetworkAclsInput{
			Filters: []*ec2.Filter{
				{
					Name:   awsgo.String("vpc-id"),
					Values: []*string{vpcID},
				}, {
					Name: awsgo.String("default"),
					Values: []*string{
						awsgo.String("true"),
					},
				},
			},
		},
	)
	if err != nil {
		logging.Debugf("[Network ACL] Failed to describe network ACL: %s", err)
		return err
	}

	if len(networkACLs.NetworkAcls) == 0 {
		logging.Debugf("[Network ACL] Nothing found to check the default association: %s", awsgo.StringValue(id))
		return nil
	}

	var defaultNetworkAclID *string
	defaultNetworkAclID = networkACLs.NetworkAcls[0].NetworkAclId

	// replace the association with default
	err = replaceNetworkAclAssociation(nacl.Client, defaultNetworkAclID, networkAcl.Associations)
	if err != nil {
		logging.Debugf("Failed to replace network ACL associations: %s", awsgo.StringValue(defaultNetworkAclID))
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
		e := report.Entry{ // Use the 'r' alias to refer to the package
			Identifier:   awsgo.StringValue(id),
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

func replaceNetworkAclAssociation(client ec2iface.EC2API, networkAclId *string, associations []*ec2.NetworkAclAssociation) error {
	logging.Debugf("Start replacing network ACL associations: %s", awsgo.StringValue(networkAclId))

	for _, association := range associations {
		logging.Debugf("Found %d network ACL associations to replace", len(associations))

		_, err := client.ReplaceNetworkAclAssociation(&ec2.ReplaceNetworkAclAssociationInput{
			AssociationId: association.NetworkAclAssociationId,
			NetworkAclId:  networkAclId,
		})
		if err != nil {
			logging.Debugf("Failed to replace network ACL association: %s to default",
				awsgo.StringValue(association.NetworkAclAssociationId))
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Successfully replaced network ACL association: %s to default",
			awsgo.StringValue(association.NetworkAclAssociationId))
	}
	logging.Debugf("Successfully replaced network ACL associations: %s", awsgo.StringValue(networkAclId))
	return nil
}

func nukeNetworkAcl(client ec2iface.EC2API, id *string) error {
	logging.Debugf("Deleting network Acl %s", awsgo.StringValue(id))

	if _, err := client.DeleteNetworkAcl(&ec2.DeleteNetworkAclInput{
		NetworkAclId: id,
	}); err != nil {
		logging.Debugf("An error happened while nuking NACL %s, error %v", awsgo.StringValue(id), err)
		return err
	}
	logging.Debugf("[Ok] network acl deleted successfully %s", awsgo.StringValue(id))

	return nil
}
