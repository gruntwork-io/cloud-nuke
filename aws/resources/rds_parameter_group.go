package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// RdsParameterGroupAPI defines the interface for RDS Parameter Group operations.
type RdsParameterGroupAPI interface {
	DeleteDBParameterGroup(ctx context.Context, params *rds.DeleteDBParameterGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBParameterGroupOutput, error)
	DescribeDBParameterGroups(ctx context.Context, params *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error)
}

// NewRdsParameterGroup creates a new RDS Parameter Group resource using the generic resource pattern.
func NewRdsParameterGroup() AwsResource {
	return NewAwsResource(&resource.Resource[RdsParameterGroupAPI]{
		ResourceTypeName: "rds-parameter-group",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[RdsParameterGroupAPI], cfg aws.Config) {
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.RdsParameterGroup
		},
		Lister: listRdsParameterGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteRdsParameterGroup),
	})
}

// listRdsParameterGroups retrieves all RDS parameter groups that match the config filters.
func listRdsParameterGroups(ctx context.Context, client RdsParameterGroupAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	// Initialize the paginator
	paginator := rds.NewDescribeDBParameterGroupsPaginator(client, &rds.DescribeDBParameterGroupsInput{})

	// Iterate through the pages
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Process each parameter group on the page
		for _, parameterGroup := range page.DBParameterGroups {
			// we can't delete default parameter group
			// Default parameter group names can include a period, such as default.mysql8.0. However, custom parameter group names can't include a period.
			if strings.HasPrefix(aws.ToString(parameterGroup.DBParameterGroupName), "default.") {
				logging.Debugf("Skipping %s since it is a default parameter group", aws.ToString(parameterGroup.DBParameterGroupName))
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: parameterGroup.DBParameterGroupName,
			}) {
				names = append(names, parameterGroup.DBParameterGroupName)
			}
		}
	}

	return names, nil
}

// deleteRdsParameterGroup deletes a single RDS parameter group.
func deleteRdsParameterGroup(ctx context.Context, client RdsParameterGroupAPI, identifier *string) error {
	_, err := client.DeleteDBParameterGroup(ctx, &rds.DeleteDBParameterGroupInput{
		DBParameterGroupName: identifier,
	})
	return err
}
