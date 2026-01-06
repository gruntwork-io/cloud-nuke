package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// EC2IPAMAPI defines the interface for EC2 IPAM operations.
type EC2IPAMAPI interface {
	DescribeIpams(ctx context.Context, params *ec2.DescribeIpamsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamsOutput, error)
	DeleteIpam(ctx context.Context, params *ec2.DeleteIpamInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamOutput, error)
	GetIpamPoolCidrs(ctx context.Context, params *ec2.GetIpamPoolCidrsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolCidrsOutput, error)
	DeprovisionIpamPoolCidr(ctx context.Context, params *ec2.DeprovisionIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.DeprovisionIpamPoolCidrOutput, error)
	GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error)
	ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error)
	DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error)
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error)
}

// NewEC2IPAM creates a new EC2 IPAM resource using the generic resource pattern.
func NewEC2IPAM() AwsResource {
	return NewAwsResource(&resource.Resource[EC2IPAMAPI]{
		ResourceTypeName: "ipam",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAM
		},
		Lister:             listEC2IPAMs,
		Nuker:              resource.SequentialDeleter(nukeEC2IPAM),
		PermissionVerifier: verifyEC2IPAMPermission,
	})
}

// listEC2IPAMs retrieves all IPAMs that match the config filters.
func listEC2IPAMs(ctx context.Context, client EC2IPAMAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	paginator := ec2.NewDescribeIpamsPaginator(client, &ec2.DescribeIpamsInput{
		MaxResults: aws.Int32(10),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, ipam := range page.Ipams {
			tagMap := util.ConvertTypesTagsToMap(ipam.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, ipam.IpamId, tagMap)
			if err != nil {
				logging.Errorf("Unable to retrieve first seen tag for IPAM %s: %v", aws.ToString(ipam.IpamId), err)
				continue
			}

			var ipamName string
			if name, ok := tagMap["Name"]; ok {
				ipamName = name
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: &ipamName,
				Time: firstSeenTime,
				Tags: tagMap,
			}) {
				result = append(result, ipam.IpamId)
			}
		}
	}

	return result, nil
}

// verifyEC2IPAMPermission performs a dry-run delete to check permissions.
func verifyEC2IPAMPermission(ctx context.Context, client EC2IPAMAPI, id *string) error {
	_, err := client.DeleteIpam(ctx, &ec2.DeleteIpamInput{
		IpamId:  id,
		Cascade: aws.Bool(true),
		DryRun:  aws.Bool(true),
	})
	return err
}

// nukeEC2IPAM deletes a single IPAM and its associated public scope pools.
func nukeEC2IPAM(ctx context.Context, client EC2IPAMAPI, id *string) error {
	// First, nuke public IPAM pools (cascade delete only handles private scope)
	if err := nukePublicIPAMPools(ctx, client, id); err != nil {
		return err
	}

	// Then delete the IPAM itself
	_, err := client.DeleteIpam(ctx, &ec2.DeleteIpamInput{
		IpamId:  id,
		Cascade: aws.Bool(true), // Deletes private scopes, pools, and allocations
	})
	return err
}

// nukePublicIPAMPools deletes all pools in the public scope of an IPAM.
// The deleteIPAM cascade option only handles private scope, so we must
// manually delete public scope pools before deleting the IPAM.
func nukePublicIPAMPools(ctx context.Context, client EC2IPAMAPI, ipamID *string) error {
	// Get the IPAM to find its public scope ID
	ipam, err := client.DescribeIpams(ctx, &ec2.DescribeIpamsInput{
		IpamIds: []string{*ipamID},
	})
	if err != nil {
		return err
	}

	if len(ipam.Ipams) == 0 {
		return nil
	}

	// Get the public scope details
	scope, err := client.DescribeIpamScopes(ctx, &ec2.DescribeIpamScopesInput{
		IpamScopeIds: []string{*ipam.Ipams[0].PublicDefaultScopeId},
	})
	if err != nil {
		return err
	}

	if len(scope.IpamScopes) == 0 {
		return nil
	}

	// Get pools in the public scope
	pools, err := client.DescribeIpamPools(ctx, &ec2.DescribeIpamPoolsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("ipam-scope-arn"),
				Values: []string{*scope.IpamScopes[0].IpamScopeArn},
			},
		},
	})
	if err != nil {
		return err
	}

	// Delete each pool after cleaning up its CIDRs and allocations
	for _, pool := range pools.IpamPools {
		if err := deProvisionPoolCIDRs(ctx, client, pool.IpamPoolId); err != nil {
			return err
		}
		if err := releaseCustomAllocations(ctx, client, pool.IpamPoolId); err != nil {
			return err
		}
		if _, err := client.DeleteIpamPool(ctx, &ec2.DeleteIpamPoolInput{
			IpamPoolId: pool.IpamPoolId,
		}); err != nil {
			return err
		}
		logging.Debugf("Deleted IPAM Pool %s from IPAM %s", aws.ToString(pool.IpamPoolId), aws.ToString(ipamID))
	}

	return nil
}

// deProvisionPoolCIDRs removes all provisioned CIDRs from a pool.
func deProvisionPoolCIDRs(ctx context.Context, client EC2IPAMAPI, poolID *string) error {
	output, err := client.GetIpamPoolCidrs(ctx, &ec2.GetIpamPoolCidrsInput{
		IpamPoolId: poolID,
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"provisioned"},
			},
		},
	})
	if err != nil {
		return err
	}

	for _, poolCidr := range output.IpamPoolCidrs {
		if _, err := client.DeprovisionIpamPoolCidr(ctx, &ec2.DeprovisionIpamPoolCidrInput{
			IpamPoolId: poolID,
			Cidr:       poolCidr.Cidr,
		}); err != nil {
			return err
		}
		logging.Debugf("De-Provisioned CIDR from IPAM Pool %s", aws.ToString(poolID))
	}

	return nil
}

// releaseCustomAllocations releases all custom allocated CIDRs from a pool.
func releaseCustomAllocations(ctx context.Context, client EC2IPAMAPI, poolID *string) error {
	output, err := client.GetIpamPoolAllocations(ctx, &ec2.GetIpamPoolAllocationsInput{
		IpamPoolId: poolID,
	})
	if err != nil {
		return err
	}

	for _, allocation := range output.IpamPoolAllocations {
		if allocation.ResourceType != types.IpamPoolAllocationResourceTypeCustom {
			continue
		}
		if _, err := client.ReleaseIpamPoolAllocation(ctx, &ec2.ReleaseIpamPoolAllocationInput{
			IpamPoolId:           poolID,
			IpamPoolAllocationId: allocation.IpamPoolAllocationId,
			Cidr:                 allocation.Cidr,
		}); err != nil {
			return err
		}
		logging.Debugf("Released custom allocated CIDR from IPAM Pool %s", aws.ToString(poolID))
	}

	return nil
}
