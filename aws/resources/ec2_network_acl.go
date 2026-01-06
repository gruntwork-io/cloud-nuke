package resources

import (
	"context"
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

// NetworkACLAPI defines the interface for Network ACL operations.
type NetworkACLAPI interface {
	DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)
	DeleteNetworkAcl(ctx context.Context, params *ec2.DeleteNetworkAclInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkAclOutput, error)
	ReplaceNetworkAclAssociation(ctx context.Context, params *ec2.ReplaceNetworkAclAssociationInput, optFns ...func(*ec2.Options)) (*ec2.ReplaceNetworkAclAssociationOutput, error)
}

// NewNetworkACL creates a new NetworkACL resource using the generic resource pattern.
func NewNetworkACL() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkACLAPI]{
		ResourceTypeName: "network-acl",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkACLAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkACL
		},
		Lister:             listNetworkACLs,
		Nuker:              resource.MultiStepDeleter(replaceNetworkACLAssociations, deleteNetworkACL),
		PermissionVerifier: verifyNetworkACLNukePermission,
	})
}

// listNetworkACLs retrieves all non-default Network ACLs that match the config filters.
func listNetworkACLs(ctx context.Context, client NetworkACLAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := ec2.NewDescribeNetworkAclsPaginator(client, &ec2.DescribeNetworkAclsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("default"),
				Values: []string{"false"},
			},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Network ACL] Failed to list network ACLs: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, networkAcl := range page.NetworkAcls {
			tagMap := util.ConvertTypesTagsToMap(networkAcl.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, networkAcl.NetworkAclId, tagMap)
			if err != nil {
				logging.Errorf("[Network ACL] Unable to retrieve first seen tag for ACL ID: %s, error: %s", *networkAcl.NetworkAclId, err)
				continue
			}

			if shouldIncludeNetworkACL(&networkAcl, firstSeenTime, cfg) {
				identifiers = append(identifiers, networkAcl.NetworkAclId)
			}
		}
	}

	return identifiers, nil
}

// shouldIncludeNetworkACL determines if a Network ACL should be included for deletion.
func shouldIncludeNetworkACL(networkAcl *types.NetworkAcl, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	var naclName string
	tagMap := util.ConvertTypesTagsToMap(networkAcl.Tags)
	if name, ok := tagMap["Name"]; ok {
		naclName = name
	}
	return cfg.ShouldInclude(config.ResourceValue{
		Name: &naclName,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

// verifyNetworkACLNukePermission performs a dry-run delete to check permissions.
func verifyNetworkACLNukePermission(ctx context.Context, client NetworkACLAPI, id *string) error {
	_, err := client.DeleteNetworkAcl(ctx, &ec2.DeleteNetworkAclInput{
		NetworkAclId: id,
		DryRun:       aws.Bool(true),
	})
	return util.TransformAWSError(err)
}

// replaceNetworkACLAssociations replaces subnet associations with the default ACL.
// Important: You can't delete a Network ACL if it's associated with any subnets.
// This step finds the default ACL for the VPC and reassigns all subnet associations to it.
func replaceNetworkACLAssociations(ctx context.Context, client NetworkACLAPI, id *string) error {
	aclID := aws.ToString(id)
	logging.Debugf("[Network ACL] Replacing associations for: %s", aclID)

	// Describe the network ACL to get its VPC and associations
	resp, err := client.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []string{aclID},
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to describe: %s", err)
		return errors.WithStackTrace(err)
	}

	if len(resp.NetworkAcls) == 0 {
		logging.Debugf("[Network ACL] Not found: %s", aclID)
		return nil
	}

	networkAcl := resp.NetworkAcls[0]
	if len(networkAcl.Associations) == 0 {
		logging.Debugf("[Network ACL] No associations to replace for: %s", aclID)
		return nil
	}

	// Get the default network ACL for this VPC
	defaultACLResp, err := client.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{
		Filters: []types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{aws.ToString(networkAcl.VpcId)}},
			{Name: aws.String("default"), Values: []string{"true"}},
		},
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to describe default ACL: %s", err)
		return errors.WithStackTrace(err)
	}

	if len(defaultACLResp.NetworkAcls) == 0 {
		logging.Debugf("[Network ACL] No default ACL found for VPC: %s", aws.ToString(networkAcl.VpcId))
		return nil
	}

	defaultACLID := defaultACLResp.NetworkAcls[0].NetworkAclId
	logging.Debugf("[Network ACL] Replacing associations with default ACL: %s", aws.ToString(defaultACLID))

	// Replace each association with the default ACL
	for _, assoc := range networkAcl.Associations {
		_, err := client.ReplaceNetworkAclAssociation(ctx, &ec2.ReplaceNetworkAclAssociationInput{
			AssociationId: assoc.NetworkAclAssociationId,
			NetworkAclId:  defaultACLID,
		})
		if err != nil {
			logging.Debugf("[Network ACL] Failed to replace association %s: %s", aws.ToString(assoc.NetworkAclAssociationId), err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("[Network ACL] Replaced association: %s", aws.ToString(assoc.NetworkAclAssociationId))
	}

	logging.Debugf("[Network ACL] Successfully replaced all associations for: %s", aclID)
	return nil
}

// deleteNetworkACL deletes a single Network ACL.
func deleteNetworkACL(ctx context.Context, client NetworkACLAPI, id *string) error {
	logging.Debugf("[Network ACL] Deleting: %s", aws.ToString(id))

	_, err := client.DeleteNetworkAcl(ctx, &ec2.DeleteNetworkAclInput{
		NetworkAclId: id,
	})
	if err != nil {
		logging.Debugf("[Network ACL] Failed to delete %s: %s", aws.ToString(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[Network ACL] Successfully deleted: %s", aws.ToString(id))
	return nil
}
