package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

// Filter all active ecs clusters
func getAllActiveEcsClusterArns(awsSession *session.Session, configObj config.Config) ([]*string, error) {
	svc := ecs.New(awsSession)

	allClusters, err := getAllEcsClusters(awsSession)
	if err != nil {
		logging.Logger.Debug("Error getting all ECS clusters")
		return nil, errors.WithStackTrace(err)
	}

	var filteredEcsClusterArns []*string

	batches := split(aws.StringValueSlice(allClusters), describeClustersRequestBatchSize)
	for _, batch := range batches {
		input := &ecs.DescribeClustersInput{
			Clusters: awsgo.StringSlice(batch),
		}

		describedClusters, describeErr := svc.DescribeClusters(input)
		if describeErr != nil {
			logging.Logger.Debugf("Error describing ECS clusters from input %s: ", input)
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
	logging.Logger.Debugf("Status for ECS Cluster %s is %s", aws.StringValue(cluster.ClusterArn), aws.StringValue(cluster.Status))
	if aws.StringValue(cluster.Status) != activeEcsClusterStatus {
		return false
	}

	return config.ShouldInclude(
		awsgo.StringValue(cluster.ClusterName),
		configObj.ECSCluster.IncludeRule.NamesRegExp,
		configObj.ECSCluster.ExcludeRule.NamesRegExp,
	)
}

func getAllEcsClustersOlderThan(awsSession *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	clusterArns, err := getAllActiveEcsClusterArns(awsSession, configObj)
	if err != nil {
		logging.Logger.Debugf("Error getting all ECS clusters with `ACTIVE` status")
		return nil, errors.WithStackTrace(err)
	}

	var filteredEcsClusters []*string
	for _, clusterArn := range clusterArns {

		firstSeenTime, err := getFirstSeenEcsClusterTag(awsSession, clusterArn)
		if err != nil {
			logging.Logger.Debugf("Error getting the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.StringValue(clusterArn))
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime.IsZero() {
			err := tagEcsClusterWhenFirstSeen(awsSession, clusterArn, time.Now().UTC())
			if err != nil {
				logging.Logger.Debugf("Error tagging the ECS cluster with ARN %s", aws.StringValue(clusterArn))
				return nil, errors.WithStackTrace(err)
			}
		} else if excludeAfter.After(firstSeenTime) {
			filteredEcsClusters = append(filteredEcsClusters, clusterArn)
		}
	}
	return filteredEcsClusters, nil
}

func nukeEcsClusters(awsSession *session.Session, ecsClusterArns []*string) error {
	svc := ecs.New(awsSession)

	numNuking := len(ecsClusterArns)

	if numNuking == 0 {
		logging.Logger.Debugf("No ECS clusters to nuke in region %s", aws.StringValue(awsSession.Config.Region))
		return nil
	}

	logging.Logger.Debugf("Deleting %d ECS clusters in region %s", numNuking, aws.StringValue(awsSession.Config.Region))

	var nukedEcsClusters []*string
	for _, clusterArn := range ecsClusterArns {
		params := &ecs.DeleteClusterInput{
			Cluster: clusterArn,
		}
		_, err := svc.DeleteCluster(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(clusterArn),
			ResourceType: "ECS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("Error, failed to delete cluster with ARN %s", aws.StringValue(clusterArn))
			return errors.WithStackTrace(err)
		}

		logging.Logger.Debugf("Success, deleted cluster: %s", aws.StringValue(clusterArn))
		nukedEcsClusters = append(nukedEcsClusters, clusterArn)
	}

	numNuked := len(nukedEcsClusters)
	logging.Logger.Debugf("[OK] %d of %d ECS cluster(s) deleted in %s", numNuked, numNuking, aws.StringValue(awsSession.Config.Region))

	return nil
}

// Tag an ECS cluster identified by the given cluster ARN when it's first seen by cloud-nuke
func tagEcsClusterWhenFirstSeen(awsSession *session.Session, clusterArn *string, timestamp time.Time) error {
	svc := ecs.New(awsSession)

	firstSeenTime := formatTimestampTag(timestamp)

	input := &ecs.TagResourceInput{
		ResourceArn: clusterArn,
		Tags: []*ecs.Tag{
			{
				Key:   aws.String(firstSeenTagKey),
				Value: aws.String(firstSeenTime),
			},
		},
	}

	_, err := svc.TagResource(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Get the `cloud-nuke-first-seen` tag value for a given ECS cluster
func getFirstSeenEcsClusterTag(awsSession *session.Session, clusterArn *string) (time.Time, error) {
	var firstSeenTime time.Time

	svc := ecs.New(awsSession)
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := svc.ListTagsForResource(input)
	if err != nil {
		logging.Logger.Debugf("Error getting the tags for ECS cluster with ARN %s", aws.StringValue(clusterArn))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range clusterTags.Tags {
		if aws.StringValue(tag.Key) == firstSeenTagKey {

			firstSeenTime, err := parseTimestampTag(aws.StringValue(tag.Value))
			if err != nil {
				logging.Logger.Debugf("Error parsing the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.StringValue(clusterArn))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}
	return firstSeenTime, nil
}

func parseTimestampTag(timestamp string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		logging.Logger.Debugf("Error parsing the timestamp into a `RFC3339` Time format")
		return parsed, errors.WithStackTrace(err)

	}
	return parsed, nil
}

func formatTimestampTag(timestamp time.Time) string {
	return timestamp.Format(time.RFC3339)
}
