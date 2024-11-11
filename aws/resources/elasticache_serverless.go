package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (cache *ElasticCacheServerless) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[ElasticCache Serverless] No cluster found in region %s", cache.Region)
		return nil
	}

	logging.Debugf("[ElasticCache Serverless] Deleting all clusters in %s", cache.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		_, err := cache.Client.DeleteServerlessCache(cache.Context, &elasticache.DeleteServerlessCacheInput{
			ServerlessCacheName: identifier,
		})
		if err != nil {
			logging.Debugf("[ElasticCache Serverless] Error deleting cluster %s in region %s", *identifier, cache.Region)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[ElasticCache Serverless] Deleted cluster %s in region %s", *identifier, cache.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: cache.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d ElasticCache Serverless deleted in %s", len(deleted), cache.Region)
	return nil
}

func (cache *ElasticCacheServerless) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var output []*string

	paginator := elasticache.NewDescribeServerlessCachesPaginator(cache.Client, &elasticache.DescribeServerlessCachesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range page.ServerlessCaches {
			if strings.ToLower(*cluster.Status) != "available" {
				logging.Debugf(
					"[ElasticCache Serverless] skiping cluster: %s, status: %s",
					*cluster.ARN,
					*cluster.Status,
				)

				continue
			}

			split := strings.Split(*cluster.ARN, ":")
			if len(split) == 0 {
				logging.Debugf(
					"[ElasticCache Serverless] skiping cluster: %s",
					*cluster.ARN,
				)

				continue
			}

			name := split[len(split)-1]

			if cnfObj.ElasticCacheServerless.ShouldInclude(config.ResourceValue{
				Name: aws.String(name),
				Time: cluster.CreateTime,
			}) {
				output = append(output, aws.String(name))
			}
		}
	}

	return output, nil
}
