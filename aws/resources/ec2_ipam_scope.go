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

// EC2IPAMScopeAPI defines the interface for EC2 IPAM Scope operations.
type EC2IPAMScopeAPI interface {
	DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error)
	DeleteIpamScope(ctx context.Context, params *ec2.DeleteIpamScopeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamScopeOutput, error)
}

// NewEC2IPAMScope creates a new EC2 IPAM Scope resource using the generic resource pattern.
func NewEC2IPAMScope() AwsResource {
	return NewAwsResource(&resource.Resource[EC2IPAMScopeAPI]{
		ResourceTypeName: "ipam-scope",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMScopeAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAMScope
		},
		Lister:             listEC2IPAMScopes,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2IPAMScope),
		PermissionVerifier: verifyEC2IPAMScopePermission,
	})
}

// listEC2IPAMScopes retrieves all non-default IPAM scopes that match the config filters.
func listEC2IPAMScopes(ctx context.Context, client EC2IPAMScopeAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	paginator := ec2.NewDescribeIpamScopesPaginator(client, &ec2.DescribeIpamScopesInput{
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

		for _, ipamScope := range page.IpamScopes {
			tagMap := util.ConvertTypesTagsToMap(ipamScope.Tags)
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, ipamScope.IpamScopeId, tagMap)
			if err != nil {
				logging.Errorf("Unable to retrieve first seen tag for IPAM Scope %s: %v", aws.ToString(ipamScope.IpamScopeId), err)
				continue
			}

			var scopeName string
			if name, ok := tagMap["Name"]; ok {
				scopeName = name
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: &scopeName,
				Time: firstSeenTime,
				Tags: tagMap,
			}) {
				result = append(result, ipamScope.IpamScopeId)
			}
		}
	}

	return result, nil
}

// verifyEC2IPAMScopePermission performs a dry-run delete to check permissions.
func verifyEC2IPAMScopePermission(ctx context.Context, client EC2IPAMScopeAPI, id *string) error {
	_, err := client.DeleteIpamScope(ctx, &ec2.DeleteIpamScopeInput{
		IpamScopeId: id,
		DryRun:      aws.Bool(true),
	})
	return err
}

// deleteEC2IPAMScope deletes a single IPAM Scope.
func deleteEC2IPAMScope(ctx context.Context, client EC2IPAMScopeAPI, id *string) error {
	_, err := client.DeleteIpamScope(ctx, &ec2.DeleteIpamScopeInput{
		IpamScopeId: id,
	})
	return err
}
