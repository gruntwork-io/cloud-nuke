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
)

// EC2SubnetAPI defines the interface for EC2 Subnet operations.
type EC2SubnetAPI interface {
	DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
}

// NewEC2Subnet creates a new EC2 Subnet resource using the generic resource pattern.
func NewEC2Subnet() AwsResource {
	return NewEC2AwsResource[EC2SubnetAPI](
		"ec2-subnet",
		WrapAwsInitClient(func(r *resource.Resource[EC2SubnetAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		func(c config.Config) config.EC2ResourceType { return c.EC2Subnet },
		listEC2Subnets,
		resource.SimpleBatchDeleter(deleteSubnet),
		&EC2ResourceOptions[EC2SubnetAPI]{PermissionVerifier: verifyEC2SubnetPermission},
	)
}

// listEC2Subnets retrieves all EC2 Subnets that match the config filters.
func listEC2Subnets(ctx context.Context, client EC2SubnetAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
	var subnetIds []*string

	// Configure filters for default subnets if requested
	var filters []types.Filter
	if defaultOnly {
		logging.Debugf("[default only] Retrieving the default subnets")
		filters = append(filters, types.Filter{
			Name:   aws.String("default-for-az"),
			Values: []string{"true"},
		})
	}

	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{
		Filters: filters,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, subnet := range page.Subnets {
			tagMap := util.ConvertTypesTagsToMap(subnet.Tags)

			// Get first seen time from tags
			firstSeenTime := getEC2SubnetFirstSeenTime(tagMap)

			if shouldIncludeEC2Subnet(subnet, firstSeenTime, cfg) {
				subnetIds = append(subnetIds, subnet.SubnetId)
			}
		}
	}

	return subnetIds, nil
}

// getEC2SubnetFirstSeenTime extracts the first seen time from tag map.
func getEC2SubnetFirstSeenTime(tagMap map[string]string) *time.Time {
	if firstSeenStr, ok := tagMap[util.FirstSeenTagKey]; ok {
		if t, err := util.ParseTimestamp(aws.String(firstSeenStr)); err == nil {
			return t
		}
	}
	return nil
}

// shouldIncludeEC2Subnet determines if a subnet should be included based on config filters.
func shouldIncludeEC2Subnet(subnet types.Subnet, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	tagMap := util.ConvertTypesTagsToMap(subnet.Tags)
	return cfg.ShouldInclude(config.ResourceValue{
		Name: util.GetEC2ResourceNameTagValue(subnet.Tags),
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// verifyEC2SubnetPermission performs a dry-run delete to check permissions.
func verifyEC2SubnetPermission(ctx context.Context, client EC2SubnetAPI, id *string) error {
	_, err := client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: id,
		DryRun:   aws.Bool(true),
	})
	return err
}

// deleteSubnet deletes a single EC2 Subnet.
func deleteSubnet(ctx context.Context, client EC2SubnetAPI, id *string) error {
	logging.Debugf("Deleting subnet %s", aws.ToString(id))
	_, err := client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: id,
	})
	return err
}
