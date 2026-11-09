package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// DBSubnetGroupsAPI defines the interface for RDS DB Subnet Group operations.
type DBSubnetGroupsAPI interface {
	AddTagsToResource(ctx context.Context, params *rds.AddTagsToResourceInput, optFns ...func(*rds.Options)) (*rds.AddTagsToResourceOutput, error)
	DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error)
	DeleteDBSubnetGroup(ctx context.Context, params *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error)
	ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
}

// NewDBSubnetGroups creates a new DBSubnetGroups resource using the generic resource pattern.
func NewDBSubnetGroups() AwsResource {
	return NewAwsResource(&resource.Resource[DBSubnetGroupsAPI]{
		ResourceTypeName: "rds-subnet-group",
		BatchSize:        DefaultBatchSize,
		FirstSeenTag:     true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DBSubnetGroupsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
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
// DB Subnet Groups do not have resource timestamps, so we use the first-seen tagging pattern.
func listDBSubnetGroups(ctx context.Context, client DBSubnetGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	excludeFirstSeenTag, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	paginator := rds.NewDescribeDBSubnetGroupsPaginator(client, &rds.DescribeDBSubnetGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, subnetGroup := range page.DBSubnetGroups {
			// Subnet groups require separate tag lookup
			tagsOutput, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: subnetGroup.DBSubnetGroupArn,
			})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			tags := util.ConvertRDSTypeTagsToMap(tagsOutput.TagList)

			// A group excluded by name or tag must not be modified by the scan.
			if !passesNameAndTagRules(cfg, subnetGroup.DBSubnetGroupName, tags) {
				continue
			}

			var firstSeenTime *time.Time
			if !excludeFirstSeenTag {
				firstSeenTime = parseDBSubnetGroupFirstSeenTag(tags)
				if firstSeenTime == nil {
					if err := setDBSubnetGroupFirstSeenTag(ctx, client, subnetGroup.DBSubnetGroupArn, time.Now().UTC()); err != nil {
						logging.Warnf(
							"Unable to tag DB subnet group %s with %s, so a time filter can never include it: %v",
							aws.ToString(subnetGroup.DBSubnetGroupName), firstSeenTagKey, err)
					}
				}
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: subnetGroup.DBSubnetGroupName,
				Time: firstSeenTime,
				Tags: tags,
			}) {
				names = append(names, subnetGroup.DBSubnetGroupName)
			}
		}
	}

	return names, nil
}

// parseDBSubnetGroupFirstSeenTag extracts the first-seen timestamp from a tags map (no API call).
func parseDBSubnetGroupFirstSeenTag(tags map[string]string) *time.Time {
	value, ok := tags[firstSeenTagKey]
	if !ok {
		return nil
	}
	parsed, err := util.ParseTimestamp(aws.String(value))
	if err != nil {
		logging.Debugf("Error parsing first-seen tag value %q: %v", value, err)
		return nil
	}
	return parsed
}

// setDBSubnetGroupFirstSeenTag tags a DB Subnet Group identified by the given ARN when it's first seen by cloud-nuke
func setDBSubnetGroupFirstSeenTag(ctx context.Context, client DBSubnetGroupsAPI, arn *string, timestamp time.Time) error {
	logging.Debugf("Tagging the DB subnet group with ARN %s with first seen timestamp", aws.ToString(arn))
	_, err := client.AddTagsToResource(ctx, &rds.AddTagsToResourceInput{
		ResourceName: arn,
		Tags: []types.Tag{
			{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(util.FormatTimestamp(timestamp)),
			},
		},
	})
	return errors.WithStackTrace(err)
}

// deleteDBSubnetGroup deletes a single RDS DB Subnet Group.
// Delete is synchronous - no wait needed.
func deleteDBSubnetGroup(ctx context.Context, client DBSubnetGroupsAPI, name *string) error {
	_, err := client.DeleteDBSubnetGroup(ctx, &rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: name,
	})
	return errors.WithStackTrace(err)
}
