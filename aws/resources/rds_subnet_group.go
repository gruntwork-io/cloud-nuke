package resources

import (
	"context"
	goerr "errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
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
		BatchSize:        49,
		InitClient: func(r *resource.Resource[DBSubnetGroupsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for RDS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = rds.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBSubnetGroups
		},
		Lister: listDBSubnetGroups,
		Nuker:  resource.SequentialDeleter(deleteDBSubnetGroup),
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
			tagsRes, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: subnetGroup.DBSubnetGroupArn,
			})
			if err != nil {
				return nil, fmt.Errorf("fail to list tags: %w", err)
			}

			rv := config.ResourceValue{
				Name: subnetGroup.DBSubnetGroupName,
				Tags: map[string]string{},
			}
			for _, v := range tagsRes.TagList {
				rv.Tags[*v.Key] = *v.Value
			}
			if cfg.ShouldInclude(rv) {
				names = append(names, subnetGroup.DBSubnetGroupName)
			}
		}
	}

	return names, nil
}

// deleteDBSubnetGroup deletes a single RDS DB Subnet Group.
func deleteDBSubnetGroup(ctx context.Context, client DBSubnetGroupsAPI, name *string) error {
	_, err := client.DeleteDBSubnetGroup(ctx, &rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: name,
	})
	if err != nil {
		return err
	}

	return waitUntilRdsDbSubnetGroupDeleted(ctx, client, name)
}

// waitUntilRdsDbSubnetGroupDeleted waits for an RDS DB Subnet Group to be deleted.
func waitUntilRdsDbSubnetGroupDeleted(ctx context.Context, client DBSubnetGroupsAPI, name *string) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := client.DescribeDBSubnetGroups(ctx, &rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: name})
		if err != nil {
			var notFoundErr *types.DBSubnetGroupNotFoundFault
			if goerr.As(err, &notFoundErr) {
				return nil
			}
			return err
		}

		time.Sleep(10 * time.Second)
		logging.Debug("Waiting for RDS DB Subnet Group to be deleted")
	}

	return RdsDeleteError{name: *name}
}
