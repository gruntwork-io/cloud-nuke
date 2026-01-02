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

// EC2IPAMResourceDiscoveryAPI defines the interface for EC2 IPAM Resource Discovery operations.
type EC2IPAMResourceDiscoveryAPI interface {
	DescribeIpamResourceDiscoveries(ctx context.Context, params *ec2.DescribeIpamResourceDiscoveriesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamResourceDiscoveriesOutput, error)
	DeleteIpamResourceDiscovery(ctx context.Context, params *ec2.DeleteIpamResourceDiscoveryInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamResourceDiscoveryOutput, error)
}

// NewEC2IPAMResourceDiscovery creates a new EC2 IPAM Resource Discovery resource using the generic resource pattern.
func NewEC2IPAMResourceDiscovery() AwsResource {
	return NewAwsResource(&resource.Resource[EC2IPAMResourceDiscoveryAPI]{
		ResourceTypeName: "ipam-resource-discovery",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMResourceDiscoveryAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAMResourceDiscovery
		},
		Lister:             listEC2IPAMResourceDiscoveries,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2IPAMResourceDiscovery),
		PermissionVerifier: verifyEC2IPAMResourceDiscoveryPermission,
	})
}

// listEC2IPAMResourceDiscoveries retrieves all non-default IPAM resource discoveries that match the config filters.
func listEC2IPAMResourceDiscoveries(ctx context.Context, client EC2IPAMResourceDiscoveryAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	paginator := ec2.NewDescribeIpamResourceDiscoveriesPaginator(client, &ec2.DescribeIpamResourceDiscoveriesInput{
		MaxResults: aws.Int32(10),
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"false"},
			},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, discovery := range page.IpamResourceDiscoveries {
			tagMap := util.ConvertTypesTagsToMap(discovery.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, discovery.IpamResourceDiscoveryId, tagMap)
			if err != nil {
				logging.Errorf("Unable to retrieve first seen tag for IPAM Resource Discovery %s: %v", aws.ToString(discovery.IpamResourceDiscoveryId), err)
				continue
			}

			var discoveryName string
			if name, ok := tagMap["Name"]; ok {
				discoveryName = name
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: &discoveryName,
				Time: firstSeenTime,
				Tags: tagMap,
			}) {
				result = append(result, discovery.IpamResourceDiscoveryId)
			}
		}
	}

	return result, nil
}

// verifyEC2IPAMResourceDiscoveryPermission performs a dry-run delete to check permissions.
func verifyEC2IPAMResourceDiscoveryPermission(ctx context.Context, client EC2IPAMResourceDiscoveryAPI, id *string) error {
	_, err := client.DeleteIpamResourceDiscovery(ctx, &ec2.DeleteIpamResourceDiscoveryInput{
		IpamResourceDiscoveryId: id,
		DryRun:                  aws.Bool(true),
	})
	return err
}

// deleteEC2IPAMResourceDiscovery deletes a single IPAM Resource Discovery.
func deleteEC2IPAMResourceDiscovery(ctx context.Context, client EC2IPAMResourceDiscoveryAPI, id *string) error {
	_, err := client.DeleteIpamResourceDiscovery(ctx, &ec2.DeleteIpamResourceDiscoveryInput{
		IpamResourceDiscoveryId: id,
	})
	return err
}
