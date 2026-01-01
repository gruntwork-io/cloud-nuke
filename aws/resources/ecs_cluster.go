package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// ECSClustersAPI defines the interface for ECS Clusters operations.
type ECSClustersAPI interface {
	DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
	DeleteCluster(ctx context.Context, params *ecs.DeleteClusterInput, optFns ...func(*ecs.Options)) (*ecs.DeleteClusterOutput, error)
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListTagsForResource(ctx context.Context, params *ecs.ListTagsForResourceInput, optFns ...func(*ecs.Options)) (*ecs.ListTagsForResourceOutput, error)
	ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	StopTask(ctx context.Context, params *ecs.StopTaskInput, optFns ...func(*ecs.Options)) (*ecs.StopTaskOutput, error)
	TagResource(ctx context.Context, params *ecs.TagResourceInput, optFns ...func(*ecs.Options)) (*ecs.TagResourceOutput, error)
}

// Used in this context to determine if the ECS Cluster is ready to be used & tagged
// For more details on other valid status values: https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Cluster
const activeEcsClusterStatus string = "ACTIVE"

// Used in this context to limit the amount of clusters passed as input to the DescribeClusters function call
// For more details on this, please read here: https://docs.aws.amazon.com/cli/latest/reference/ecs/describe-clusters.html#options
const describeClustersRequestBatchSize = 100

// NewECSClusters creates a new ECS Clusters resource using the generic resource pattern.
func NewECSClusters() AwsResource {
	return NewAwsResource(&resource.Resource[ECSClustersAPI]{
		ResourceTypeName: "ecscluster",
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[ECSClustersAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for ECS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ecs.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ECSCluster
		},
		Lister: listECSClusters,
		Nuker:  deleteECSClusters,
	})
}

// listECSClusters retrieves all ECS clusters that match the config filters.
func listECSClusters(ctx context.Context, client ECSClustersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Get all cluster ARNs
	allClusters, err := getAllEcsClusters(ctx, client)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	excludeFirstSeenTag, err := util.GetBoolFromContext(ctx, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var result []*string
	clusterList := aws.ToStringSlice(allClusters)
	batches := util.Split(clusterList, describeClustersRequestBatchSize)

	for _, batch := range batches {
		resp, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: batch,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range resp.Clusters {
			if cluster.Status == nil || aws.ToString(cluster.Status) != activeEcsClusterStatus {
				continue
			}

			// Get all tags for the cluster for filtering purposes
			tags, err := getAllEcsClusterTags(ctx, client, cluster.ClusterArn)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if !cfg.ShouldInclude(config.ResourceValue{
				Name: cluster.ClusterName,
				Tags: tags,
			}) {
				continue
			}

			if excludeFirstSeenTag {
				result = append(result, cluster.ClusterArn)
				continue
			}

			firstSeenTime, err := getEcsClusterFirstSeenTag(ctx, client, cluster.ClusterArn)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if firstSeenTime == nil {
				if err := setEcsClusterFirstSeenTag(ctx, client, cluster.ClusterArn, time.Now().UTC()); err != nil {
					return nil, errors.WithStackTrace(err)
				}
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Time: firstSeenTime,
				Name: cluster.ClusterName,
				Tags: tags,
			}) {
				result = append(result, cluster.ClusterArn)
			}
		}
	}

	return result, nil
}

// getAllEcsClusters returns all ECS Cluster ARNs.
// Handles pagination until all pages are retrieved.
func getAllEcsClusters(ctx context.Context, client ECSClustersAPI) ([]*string, error) {
	var clusterArns []string
	nextToken := (*string)(nil)

	for {
		resp, err := client.ListClusters(ctx, &ecs.ListClustersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		clusterArns = append(clusterArns, resp.ClusterArns...)
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		nextToken = resp.NextToken
	}

	return aws.StringSlice(clusterArns), nil
}

// getAllEcsClusterTags retrieves all tags for a given ECS cluster and returns them as a map
func getAllEcsClusterTags(ctx context.Context, client ECSClustersAPI, clusterArn *string) (map[string]string, error) {
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := client.ListTagsForResource(ctx, input)
	if err != nil {
		logging.Debugf("Error getting the tags for ECS cluster with ARN %s", aws.ToString(clusterArn))
		return nil, errors.WithStackTrace(err)
	}

	tags := make(map[string]string)
	for _, tag := range clusterTags.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return tags, nil
}

// getEcsClusterFirstSeenTag gets the `cloud-nuke-first-seen` tag value for a given ECS cluster
func getEcsClusterFirstSeenTag(ctx context.Context, client ECSClustersAPI, clusterArn *string) (*time.Time, error) {
	var firstSeenTime *time.Time

	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := client.ListTagsForResource(ctx, input)
	if err != nil {
		logging.Debugf("Error getting the tags for ECS cluster with ARN %s", aws.ToString(clusterArn))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range clusterTags.Tags {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err = util.ParseTimestamp(tag.Value)
			if err != nil {
				logging.Debugf("Error parsing the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.ToString(clusterArn))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return firstSeenTime, nil
}

// setEcsClusterFirstSeenTag tags an ECS cluster with the first seen timestamp
func setEcsClusterFirstSeenTag(ctx context.Context, client ECSClustersAPI, clusterArn *string, timestamp time.Time) error {
	firstSeenTime := util.FormatTimestamp(timestamp)

	input := &ecs.TagResourceInput{
		ResourceArn: clusterArn,
		Tags: []types.Tag{
			{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(firstSeenTime),
			},
		},
	}

	_, err := client.TagResource(ctx, input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// stopClusterRunningTasks stops all running tasks on a cluster
func stopClusterRunningTasks(ctx context.Context, client ECSClustersAPI, clusterArn *string) error {
	logging.Debugf("[TASK] stopping tasks running on cluster %v", *clusterArn)
	// before deleting the cluster, remove the active tasks on that cluster
	runningTasks, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:       clusterArn,
		DesiredStatus: types.DesiredStatusRunning,
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	// stop the listed tasks
	for _, task := range runningTasks.TaskArns {
		_, err := client.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: clusterArn,
			Task:    aws.String(task),
			Reason:  aws.String("Terminating task due to cluster deletion"),
		})
		if err != nil {
			logging.Debugf("[TASK] Unable to stop the task %s on cluster %s. Reason: %v", task, *clusterArn, err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("[TASK] Success, stopped task %v", task)
	}
	return nil
}

// deleteECSClusters is a custom nuker function for ECS clusters.
// ECS clusters require stopping running tasks before deletion.
func deleteECSClusters(ctx context.Context, client ECSClustersAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	numNuking := len(identifiers)

	if numNuking == 0 {
		logging.Debugf("No ECS clusters to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting %d ECS clusters in region %s", numNuking, scope.Region)

	var allErrs *multierror.Error
	var nukedEcsClusters []*string
	for _, clusterArn := range identifiers {
		// before nuking the clusters, do check active tasks on the cluster and stop all of them
		err := stopClusterRunningTasks(ctx, client, clusterArn)
		if err != nil {
			logging.Debugf("Error, unable to stop the running tasks on the cluster %s %s", aws.ToString(clusterArn), err)
			// Record status of this resource
			report.Record(report.Entry{
				Identifier:   aws.ToString(clusterArn),
				ResourceType: resourceType,
				Error:        err,
			})
			allErrs = multierror.Append(allErrs, errors.WithStackTrace(err))
			continue
		}

		params := &ecs.DeleteClusterInput{
			Cluster: clusterArn,
		}
		_, err = client.DeleteCluster(ctx, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(clusterArn),
			ResourceType: resourceType,
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("Error, failed to delete cluster with ARN %s %s", aws.ToString(clusterArn), err)
			allErrs = multierror.Append(allErrs, errors.WithStackTrace(err))
			continue
		}

		logging.Debugf("Success, deleted cluster: %s", aws.ToString(clusterArn))
		nukedEcsClusters = append(nukedEcsClusters, clusterArn)
	}

	numNuked := len(nukedEcsClusters)
	logging.Debugf("[OK] %d of %d ECS cluster(s) deleted in %s", numNuked, numNuking, scope.Region)

	return allErrs.ErrorOrNil()
}
