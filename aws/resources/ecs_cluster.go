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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// Used in this context to determine if the ECS Cluster is ready to be used & tagged
// For more details on other valid status values: https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Cluster
const activeEcsClusterStatus string = "ACTIVE"

// Used in this context to limit the amount of clusters passed as input to the DescribeClusters function call
// For more details on this, please read here: https://docs.aws.amazon.com/cli/latest/reference/ecs/describe-clusters.html#options
const describeClustersRequestBatchSize = 100

// getAllEcsClusters returns all ECS Cluster ARNs.
// Handles pagination until all pages are retrieved.
func (clusters *ECSClusters) getAllEcsClusters() ([]*string, error) {
	var clusterArns []string
	nextToken := (*string)(nil)

	for {
		resp, err := clusters.Client.ListClusters(clusters.Context, &ecs.ListClustersInput{
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

func (clusters *ECSClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	allClusters, err := clusters.getAllEcsClusters()
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	excludeFirstSeenTag, err := util.GetBoolFromContext(c, util.ExcludeFirstSeenTagKey)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var result []*string
	clusterList := aws.ToStringSlice(allClusters)
	batches := util.Split(clusterList, describeClustersRequestBatchSize)

	for _, batch := range batches {
		resp, err := clusters.Client.DescribeClusters(clusters.Context, &ecs.DescribeClustersInput{
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
			tags, err := clusters.getAllTags(cluster.ClusterArn)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if !configObj.ECSCluster.ShouldInclude(config.ResourceValue{
				Name: cluster.ClusterName,
				Tags: tags,
			}) {
				continue
			}

			if excludeFirstSeenTag {
				result = append(result, cluster.ClusterArn)
				continue
			}

			firstSeenTime, err := clusters.getFirstSeenTag(cluster.ClusterArn)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if firstSeenTime == nil {
				if err := clusters.setFirstSeenTag(cluster.ClusterArn, time.Now().UTC()); err != nil {
					return nil, errors.WithStackTrace(err)
				}
				continue
			}

			if configObj.ECSCluster.ShouldInclude(config.ResourceValue{
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

func (clusters *ECSClusters) stopClusterRunningTasks(clusterArn *string) error {
	logging.Debugf("[TASK] stopping tasks running on cluster %v", *clusterArn)
	// before deleting the cluster, remove the active tasks on that cluster
	runningTasks, err := clusters.Client.ListTasks(clusters.Context, &ecs.ListTasksInput{
		Cluster:       clusterArn,
		DesiredStatus: types.DesiredStatusRunning,
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	// stop the listed tasks
	for _, task := range runningTasks.TaskArns {
		_, err := clusters.Client.StopTask(clusters.Context, &ecs.StopTaskInput{
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

func (clusters *ECSClusters) nukeAll(ecsClusterArns []*string) error {
	numNuking := len(ecsClusterArns)

	if numNuking == 0 {
		logging.Debugf("No ECS clusters to nuke in region %s", clusters.Region)
		return nil
	}

	logging.Debugf("Deleting %d ECS clusters in region %s", numNuking, clusters.Region)

	var nukedEcsClusters []*string
	for _, clusterArn := range ecsClusterArns {

		// before nuking the clusters, do check active tasks on the cluster and stop all of them
		err := clusters.stopClusterRunningTasks(clusterArn)
		if err != nil {
			logging.Debugf("Error, unable to stop the running stasks on the cluster %s %s", aws.ToString(clusterArn), err)
			return errors.WithStackTrace(err)
		}

		params := &ecs.DeleteClusterInput{
			Cluster: clusterArn,
		}
		_, err = clusters.Client.DeleteCluster(clusters.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(clusterArn),
			ResourceType: "ECS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("Error, failed to delete cluster with ARN %s %s", aws.ToString(clusterArn), err)
			return errors.WithStackTrace(err)
		}

		logging.Debugf("Success, deleted cluster: %s", aws.ToString(clusterArn))
		nukedEcsClusters = append(nukedEcsClusters, clusterArn)
	}

	numNuked := len(nukedEcsClusters)
	logging.Debugf("[OK] %d of %d ECS cluster(s) deleted in %s", numNuked, numNuking, clusters.Region)

	return nil
}

// Tag an ECS cluster identified by the given cluster ARN when it's first seen by cloud-nuke
func (clusters *ECSClusters) setFirstSeenTag(clusterArn *string, timestamp time.Time) error {
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

	_, err := clusters.Client.TagResource(clusters.Context, input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// getAllTags retrieves all tags for a given ECS cluster and returns them as a map
func (clusters *ECSClusters) getAllTags(clusterArn *string) (map[string]string, error) {
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := clusters.Client.ListTagsForResource(clusters.Context, input)
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

// Get the `cloud-nuke-first-seen` tag value for a given ECS cluster
func (clusters *ECSClusters) getFirstSeenTag(clusterArn *string) (*time.Time, error) {
	var firstSeenTime *time.Time

	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := clusters.Client.ListTagsForResource(clusters.Context, input)
	if err != nil {
		logging.Debugf("Error getting the tags for ECS cluster with ARN %s", aws.ToString(clusterArn))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range clusterTags.Tags {
		if util.IsFirstSeenTag(tag.Key) {

			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				logging.Debugf("Error parsing the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.ToString(clusterArn))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return firstSeenTime, nil
}
