package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// RdsProxyAPI defines the interface for RDS Proxy operations.
type RdsProxyAPI interface {
	DescribeDBProxies(ctx context.Context, params *rds.DescribeDBProxiesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBProxiesOutput, error)
	DeleteDBProxy(ctx context.Context, params *rds.DeleteDBProxyInput, optFns ...func(*rds.Options)) (*rds.DeleteDBProxyOutput, error)
	ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
}

// NewRdsProxy creates a new RdsProxy resource using the generic resource pattern.
func NewRdsProxy() AwsResource {
	return NewAwsResource(&resource.Resource[RdsProxyAPI]{
		ResourceTypeName: "rds-proxy",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[RdsProxyAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.RDSProxy
		},
		Lister: listRdsProxies,
		Nuker:  resource.SimpleBatchDeleter(deleteRdsProxy),
	})
}

// listRdsProxies retrieves all RDS Proxies that match the config filters.
func listRdsProxies(ctx context.Context, client RdsProxyAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string
	paginator := rds.NewDescribeDBProxiesPaginator(client, &rds.DescribeDBProxiesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, proxy := range page.DBProxies {
			tags, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: proxy.DBProxyArn,
			})
			if err != nil {
				logging.Debugf("Failed to fetch tags for RDS proxy %s: %s", aws.ToString(proxy.DBProxyName), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: proxy.DBProxyName,
				Time: proxy.CreatedDate,
				Tags: util.ConvertRDSTypeTagsToMap(tags.TagList),
			}) {
				names = append(names, proxy.DBProxyName)
			}
		}
	}

	return names, nil
}

// deleteRdsProxy deletes a single RDS Proxy.
func deleteRdsProxy(ctx context.Context, client RdsProxyAPI, name *string) error {
	_, err := client.DeleteDBProxy(ctx, &rds.DeleteDBProxyInput{
		DBProxyName: name,
	})
	return err
}
