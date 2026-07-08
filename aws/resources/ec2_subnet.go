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

// EC2SubnetAPI defines the interface for EC2 Subnet operations.
type EC2SubnetAPI interface {
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
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
			// Skip default subnets when not explicitly targeting them.
			// The defaults-aws command sets defaultOnly=true to target these;
			// the regular aws command should leave them alone.
			if !defaultOnly && aws.ToBool(subnet.DefaultForAz) {
				continue
			}

			// Subnets have no creation timestamp, so age them via the
			// first-seen tag, stamping it here on first scan.
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, subnet.SubnetId, util.ConvertTypesTagsToMap(subnet.Tags))
			if err != nil {
				logging.Error("unable to retrieve first seen tag")
				return nil, errors.WithStackTrace(err)
			}

			if shouldIncludeEC2Subnet(subnet, firstSeenTime, cfg) {
				subnetIds = append(subnetIds, subnet.SubnetId)
			}
		}
	}

	return subnetIds, nil
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
	return util.TransformAWSError(err)
}

// deleteSubnet deletes a single EC2 Subnet.
func deleteSubnet(ctx context.Context, client EC2SubnetAPI, id *string) error {
	logging.Debugf("Deleting subnet %s", aws.ToString(id))
	_, err := client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: id,
	})
	return err
}
