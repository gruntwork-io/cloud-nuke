package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// EIPAddressesAPI defines the interface for Elastic IP operations.
type EIPAddressesAPI interface {
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewEIPAddresses creates a new Elastic IP resource using the generic resource pattern.
func NewEIPAddresses() AwsResource {
	return NewAwsResource(&resource.Resource[EIPAddressesAPI]{
		ResourceTypeName: "eip",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EIPAddressesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticIP
		},
		Lister:             listEIPAddresses,
		Nuker:              resource.SimpleBatchDeleter(releaseEIPAddress),
		PermissionVerifier: verifyEIPAddressPermission,
	})
}

// listEIPAddresses retrieves all Elastic IP addresses that match the config filters.
func listEIPAddresses(ctx context.Context, client EIPAddressesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, err
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, address.AllocationId, util.ConvertTypesTagsToMap(address.Tags))
		if err != nil {
			logging.Errorf("Unable to retrieve tags for EIP %s: %v", aws.ToString(address.AllocationId), err)
			return nil, err
		}

		// If Name is unset, GetEC2ResourceNameTagValue returns nil
		allocationName := util.GetEC2ResourceNameTagValue(address.Tags)
		if cfg.ShouldInclude(config.ResourceValue{
			Time: firstSeenTime,
			Name: allocationName,
			Tags: util.ConvertTypesTagsToMap(address.Tags),
		}) {
			allocationIds = append(allocationIds, address.AllocationId)
		}
	}

	return allocationIds, nil
}

// releaseEIPAddress releases a single Elastic IP address.
func releaseEIPAddress(ctx context.Context, client EIPAddressesAPI, allocationId *string) error {
	_, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: allocationId,
	})
	return err
}

// verifyEIPAddressPermission performs a dry-run release to check permissions.
func verifyEIPAddressPermission(ctx context.Context, client EIPAddressesAPI, allocationId *string) error {
	_, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: allocationId,
		DryRun:       aws.Bool(true),
	})
	return err
}
