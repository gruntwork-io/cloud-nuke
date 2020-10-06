package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Tag an ECS cluster identified by the given cluster ARN with a tag that has the given key and value
func tagEcsCluster(awsSession *session.Session, clusterArn *string, tagKey string, tagValue string) error {
	svc := ecs.New(awsSession)
	input := &ecs.TagResourceInput{
		ResourceArn: clusterArn,
		Tags: []*ecs.Tag{
			{
				Key:   aws.String(tagKey),
				Value: aws.String(tagValue),
			},
		},
	}

	_, err := svc.TagResource(input)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func getAllEcsClustersOlderThan(awsSession *session.Session, region string, timeMargin time.Time) ([]*string, error) {
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		logging.Logger.Errorf("Error getting all ECS clusters")
		return nil, err
	}

	var filteredEcsClusters []*string
	for _, clusterArn := range clusterArns {

		firstSeenTime, err := getClusterTag(awsSession, clusterArn, "first_seen")
		if err != nil {
			logging.Logger.Errorf("Error getting the `first_seen` ECS cluster")
			return nil, err
		}

		if firstSeenTime.IsZero() {
			err := tagEcsCluster(awsSession, clusterArn, "first_seen", time.Now().UTC().String())
			if err != nil {
				logging.Logger.Errorf("Error tagigng the ECS cluster")
				return nil, err
			}
		}

		if firstSeenTime.Before(timeMargin) {
			filteredEcsClusters = append(filteredEcsClusters, clusterArn)
		}
	}

	return filteredEcsClusters, nil
}

func nukeEcsClusters(awsSession *session.Session, ecsClusterArns []*string) ([]*string, []*string) {
	svc := ecs.New(awsSession)

	var nukedEcsClusters []*string
	var failedEcsClusters []*string

	for _, clusterArn := range ecsClusterArns {
		params := &ecs.DeleteClusterInput{
			Cluster: clusterArn,
		}

		_, err := svc.DeleteCluster(params)

		if err != nil {
			logging.Logger.Errorf("Error, failed to delete cluster")
			failedEcsClusters = append(failedEcsClusters, clusterArn)
		} else {
			logging.Logger.Infof("Success, deleted cluster: %s", *clusterArn)
			nukedEcsClusters = append(nukedEcsClusters, clusterArn)
		}
	}

	return nukedEcsClusters, failedEcsClusters
}

func getClusterTag(awsSession *session.Session, clusterArn *string, tagKey string) (time.Time, error) {
	var firstSeenTime time.Time

	svc := ecs.New(awsSession)
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := svc.ListTagsForResource(input)
	if err != nil {
		logging.Logger.Errorf("Error getting the tags")
		return firstSeenTime, nil
	}

	for _, tag := range clusterTags.Tags {
		if aws.StringValue(tag.Key) == tagKey {

			firstSeenTime, err := time.Parse(time.RFC3339, *tag.Value)
			if err != nil {
				logging.Logger.Errorf("Error while tagging the ECS cluster")
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}
	return firstSeenTime, nil
}
