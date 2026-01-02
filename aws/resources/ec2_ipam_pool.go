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

// EC2IPAMPoolAPI defines the interface for EC2 IPAM Pool operations.
type EC2IPAMPoolAPI interface {
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error)
}

// NewEC2IPAMPool creates a new EC2 IPAM Pool resource using the generic resource pattern.
func NewEC2IPAMPool() AwsResource {
	return NewAwsResource(&resource.Resource[EC2IPAMPoolAPI]{
		ResourceTypeName: "ipam-pool",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMPoolAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAMPool
		},
		Lister:             listEC2IPAMPools,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2IPAMPool),
		PermissionVerifier: verifyEC2IPAMPoolPermission,
	})
}

// listEC2IPAMPools retrieves all IPAM pools in "create-complete" state that match the config filters.
func listEC2IPAMPools(ctx context.Context, client EC2IPAMPoolAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	paginator := ec2.NewDescribeIpamPoolsPaginator(client, &ec2.DescribeIpamPoolsInput{
		MaxResults: aws.Int32(10),
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"create-complete"},
			},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, pool := range page.IpamPools {
			tagMap := util.ConvertTypesTagsToMap(pool.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, pool.IpamPoolId, tagMap)
			if err != nil {
				logging.Errorf("Unable to retrieve first seen tag for IPAM Pool %s: %v", aws.ToString(pool.IpamPoolId), err)
				continue
			}

			var poolName string
			if name, ok := tagMap["Name"]; ok {
				poolName = name
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: &poolName,
				Time: firstSeenTime,
				Tags: tagMap,
			}) {
				result = append(result, pool.IpamPoolId)
			}
		}
	}

	return result, nil
}

// verifyEC2IPAMPoolPermission performs a dry-run delete to check permissions.
func verifyEC2IPAMPoolPermission(ctx context.Context, client EC2IPAMPoolAPI, id *string) error {
	_, err := client.DeleteIpamPool(ctx, &ec2.DeleteIpamPoolInput{
		IpamPoolId: id,
		DryRun:     aws.Bool(true),
	})
	return err
}

// deleteEC2IPAMPool deletes a single IPAM Pool.
func deleteEC2IPAMPool(ctx context.Context, client EC2IPAMPoolAPI, id *string) error {
	_, err := client.DeleteIpamPool(ctx, &ec2.DeleteIpamPoolInput{
		IpamPoolId: id,
	})
	return err
}
