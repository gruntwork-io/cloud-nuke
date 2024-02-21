package resources

import (
	"context"
	"time"

	"github.com/gruntwork-io/cloud-nuke/util"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Used in this context to determine if the ECS Cluster is ready to be used & tagged
// For more details on other valid status values: https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Cluster
const activeEcsClusterStatus string = "ACTIVE"

// Used in this context to limit the amount of clusters passed as input to the DescribeClusters function call
// For more details on this, please read here: https://docs.aws.amazon.com/cli/latest/reference/ecs/describe-clusters.html#options
const describeClustersRequestBatchSize = 100

// getAllEcsClusters - Returns a string of ECS Cluster ARNs, which uniquely identifies the cluster.
// We need to get all clusters before we can get all services.
func (clusters *ECSClusters) getAllEcsClusters() ([]*string, error) {
	clusterArns := []*string{}
	result, err := clusters.Client.ListClusters(&ecs.ListClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	clusterArns = append(clusterArns, result.ClusterArns...)

	// Handle pagination: continuously pull the next page if nextToken is set
	for awsgo.StringValue(result.NextToken) != "" {
		result, err = clusters.Client.ListClusters(&ecs.ListClustersInput{NextToken: result.NextToken})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		clusterArns = append(clusterArns, result.ClusterArns...)
	}

	return clusterArns, nil
}

// Filter all active ecs clusters
func (clusters *ECSClusters) getAllActiveEcsClusterArns(configObj config.Config) ([]*string, error) {
	allClusters, err := clusters.getAllEcsClusters()
	if err != nil {
		logging.Debug("Error getting all ECS clusters")
		return nil, errors.WithStackTrace(err)
	}

	var filteredEcsClusterArns []*string

	batches := util.Split(aws.StringValueSlice(allClusters), describeClustersRequestBatchSize)
	for _, batch := range batches {
		input := &ecs.DescribeClustersInput{
			Clusters: awsgo.StringSlice(batch),
		}

		describedClusters, describeErr := clusters.Client.DescribeClusters(input)
		if describeErr != nil {
			logging.Debugf("Error describing ECS clusters from input %s: ", input)
			return nil, errors.WithStackTrace(describeErr)
		}

		for _, cluster := range describedClusters.Clusters {
			if shouldIncludeECSCluster(cluster, configObj) {
				filteredEcsClusterArns = append(filteredEcsClusterArns, cluster.ClusterArn)
			}
		}
	}

	return filteredEcsClusterArns, nil
}

func shouldIncludeECSCluster(cluster *ecs.Cluster, configObj config.Config) bool {
	if cluster == nil {
		return false
	}

	// Filter out invalid state ECS Clusters (will return only `ACTIVE` state clusters)
	// `cloud-nuke` needs to tag ECS Clusters it sees for the first time.
	// Therefore to tag a cluster, that cluster must be in the `ACTIVE` state.
	logging.Debugf("Status for ECS Cluster %s is %s", aws.StringValue(cluster.ClusterArn), aws.StringValue(cluster.Status))
	if aws.StringValue(cluster.Status) != activeEcsClusterStatus {
		return false
	}

	return configObj.ECSCluster.ShouldInclude(config.ResourceValue{Name: cluster.ClusterName})
}

func (clusters *ECSClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	clusterArns, err := clusters.getAllActiveEcsClusterArns(configObj)
	if err != nil {
		logging.Debugf("Error getting all ECS clusters with `ACTIVE` status")
		return nil, errors.WithStackTrace(err)
	}

	var filteredEcsClusters []*string
	for _, clusterArn := range clusterArns {

		firstSeenTime, err := clusters.getFirstSeenTag(clusterArn)
		if err != nil {
			logging.Debugf("Error getting the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.StringValue(clusterArn))
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			err := clusters.setFirstSeenTag(clusterArn, time.Now().UTC())
			if err != nil {
				logging.Debugf("Error tagging the ECS cluster with ARN %s", aws.StringValue(clusterArn))
				return nil, errors.WithStackTrace(err)
			}
		} else if configObj.ECSCluster.ShouldInclude(config.ResourceValue{Time: firstSeenTime}) {
			filteredEcsClusters = append(filteredEcsClusters, clusterArn)
		}
	}
	return filteredEcsClusters, nil
}

func (clusters *ECSClusters) stopClusterRunningTasks(clusterArn *string) error {
	logging.Debugf("stopping tasks running on cluster %v", *clusterArn)
	// before deleting the cluster, remove the active tasks on that cluster
	runningTasks, err := clusters.Client.ListTasks(&ecs.ListTasksInput{
		Cluster:       clusterArn,
		DesiredStatus: aws.String("RUNNING"),
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	// stop the listed tasks
	for _, task := range runningTasks.TaskArns {
		_, err := clusters.Client.StopTask(&ecs.StopTaskInput{
			Cluster: clusterArn,
			Task:    task,
			Reason:  aws.String("cluster is going to be deleted"),
		})
		if err != nil {
			logging.Debugf("Unable to stop the task %s on cluster %s , Reason : %v", *task, *clusterArn, err)
		}
		logging.Debugf("task %v was stopped", *task)
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
			logging.Debugf("Error, unable to stop the running stasks on the cluster %s %s", aws.StringValue(clusterArn), err)
			return errors.WithStackTrace(err)
		}

		params := &ecs.DeleteClusterInput{
			Cluster: clusterArn,
		}
		_, err = clusters.Client.DeleteCluster(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(clusterArn),
			ResourceType: "ECS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("Error, failed to delete cluster with ARN %s %s", aws.StringValue(clusterArn), err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking ECS Cluster",
			}, map[string]interface{}{
				"region": clusters.Region,
			})
			return errors.WithStackTrace(err)
		}

		logging.Debugf("Success, deleted cluster: %s", aws.StringValue(clusterArn))
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
		Tags: []*ecs.Tag{
			{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(firstSeenTime),
			},
		},
	}

	_, err := clusters.Client.TagResource(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Get the `cloud-nuke-first-seen` tag value for a given ECS cluster
func (clusters *ECSClusters) getFirstSeenTag(clusterArn *string) (*time.Time, error) {
	var firstSeenTime *time.Time

	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := clusters.Client.ListTagsForResource(input)
	if err != nil {
		logging.Debugf("Error getting the tags for ECS cluster with ARN %s", aws.StringValue(clusterArn))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range clusterTags.Tags {
		if util.IsFirstSeenTag(tag.Key) {

			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				logging.Debugf("Error parsing the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.StringValue(clusterArn))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}

	return firstSeenTime, nil
}
