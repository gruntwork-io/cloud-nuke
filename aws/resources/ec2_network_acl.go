package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (nacl *NetworkACL) setFirstSeenTag(networkAcl ec2.NetworkAcl, value time.Time) error {
	_, err := nacl.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{networkAcl.NetworkAclId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(util.FirstSeenTagKey),
				Value: aws.String(util.FormatTimestamp(value)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (nacl *NetworkACL) getFirstSeenTag(networkAcl ec2.NetworkAcl) (*time.Time, error) {
	for _, tag := range networkAcl.Tags {
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

func (nacl *NetworkACL) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	resp, err := nacl.Client.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("default"),
				Values: []*string{
					aws.String("false"), // can't able to nuke default nacl
				},
			},
		},
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to list network ACL: %s", err)
		return nil, err
	}

	for _, networkAcl := range resp.NetworkAcls {
		// check first seen tag
		firstSeenTime, err := nacl.getFirstSeenTag(*networkAcl)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for network acl: %s, with error: %s", aws.StringValue(networkAcl.NetworkAclId), err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := nacl.setFirstSeenTag(*networkAcl, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag network acl: %s, with error: %s", aws.StringValue(networkAcl.NetworkAclId), err)
				continue
			}
		}

		if shouldIncludeNetworkACL(networkAcl, firstSeenTime, configObj) {
			identifiers = append(identifiers, networkAcl.NetworkAclId)
		}
	}

	// Check and verify the list of allowed nuke actions
	nacl.VerifyNukablePermissions(identifiers, func(id *string) error {
		_, err := nacl.Client.DeleteNetworkAcl(&ec2.DeleteNetworkAclInput{
			NetworkAclId: id,
			DryRun:       aws.Bool(true),
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
	if err := nacl.nukeNetworkAcl(id); err != nil {
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
		logging.Debugf("[Network ACL] Nothing found: %s", aws.StringValue(id))
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
					Name:   aws.String("vpc-id"),
					Values: []*string{vpcID},
				}, {
					Name: aws.String("default"),
					Values: []*string{
						aws.String("true"),
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
		logging.Debugf("[Network ACL] Nothing found to check the default association: %s", aws.StringValue(id))
		return nil
	}

	var defaultNetworkAclID *string
	defaultNetworkAclID = networkACLs.NetworkAcls[0].NetworkAclId

	// repalce the association with default
	for _, association := range networkAcl.Associations {
		logging.Debug(fmt.Sprintf("Found %d network ACL associations to replace", len(networkAcl.Associations)))
		_, err := nacl.Client.ReplaceNetworkAclAssociation(&ec2.ReplaceNetworkAclAssociationInput{
			AssociationId: association.NetworkAclAssociationId,
			NetworkAclId:  defaultNetworkAclID,
		})
		if err != nil {
			logging.Debug(fmt.Sprintf("Failed to replace network ACL association: %s to default",
				aws.StringValue(association.NetworkAclAssociationId)))
			return errors.WithStackTrace(err)
		}
		logging.Debug(fmt.Sprintf("Successfully replaced network ACL association: %s to default", aws.StringValue(association.NetworkAclAssociationId)))
	}
	return nil
}

func (nacl *NetworkACL) nukeNetworkAcl(id *string) error {
	logging.Debugf("Deleting network Acl %s", aws.StringValue(id))

	if _, err := nacl.Client.DeleteNetworkAcl(&ec2.DeleteNetworkAclInput{
		NetworkAclId: id,
	}); err != nil {
		logging.Debugf("An error happened while nuking NACL %s, error %v", aws.StringValue(id), err)
		return err
	}
	logging.Debugf("[Ok] network acl deleted successfully %s", aws.StringValue(id))

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
		if nukable, err := nacl.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		err := nacl.nuke(id)

		// Record status of this resource
		e := report.Entry{ // Use the 'r' alias to refer to the package
			Identifier:   aws.StringValue(id),
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
