package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// DBSubnetGroupsAPI defines the interface for RDS DB Subnet Group operations.
type DBSubnetGroupsAPI interface {
	DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error)
	DeleteDBSubnetGroup(ctx context.Context, params *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error)
	ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
}

// NewDBSubnetGroups creates a new DBSubnetGroups resource using the generic resource pattern.
func NewDBSubnetGroups() AwsResource {
	return NewAwsResource(&resource.Resource[DBSubnetGroupsAPI]{
		ResourceTypeName: "rds-subnet-group",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DBSubnetGroupsAPI], cfg aws.Config) {
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBSubnetGroups
		},
		Lister: listDBSubnetGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteDBSubnetGroup),
	})
}

// listDBSubnetGroups retrieves all RDS DB Subnet Groups that match the config filters.
func listDBSubnetGroups(ctx context.Context, client DBSubnetGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string
	paginator := rds.NewDescribeDBSubnetGroupsPaginator(client, &rds.DescribeDBSubnetGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, subnetGroup := range page.DBSubnetGroups {
			// Subnet groups require separate tag lookup
			tags, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: subnetGroup.DBSubnetGroupArn,
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: subnetGroup.DBSubnetGroupName,
				Tags: util.ConvertRDSTypeTagsToMap(tags.TagList),
			}) {
				names = append(names, subnetGroup.DBSubnetGroupName)
			}
		}
	}

	return names, nil
}

// deleteDBSubnetGroup deletes a single RDS DB Subnet Group.
// Delete is synchronous - no wait needed.
func deleteDBSubnetGroup(ctx context.Context, client DBSubnetGroupsAPI, name *string) error {
	_, err := client.DeleteDBSubnetGroup(ctx, &rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: name,
	})
	return errors.WithStackTrace(err)
}
