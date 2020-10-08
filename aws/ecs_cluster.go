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

// Tag an ECS cluster identified by the given cluster ARN when it's first seen by cloud-nuke
func tagEcsClusterWhenFirstSeen(awsSession *session.Session, clusterArn *string, tagKey string, tagValue string) error {
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

func getAllEcsClustersOlderThan(awsSession *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		logging.Logger.Errorf("Error getting all ECS clusters")
		return nil, errors.WithStackTrace(err)
	}

	var filteredEcsClusters []*string
	for _, clusterArn := range clusterArns {

		firstSeenTime, err := getClusterTag(awsSession, clusterArn, firstSeenTagKey)
		if err != nil {
			logging.Logger.Errorf("Error getting the `cloud-nuke-first-seen` tag for ECS cluster with ARN %s", aws.StringValue(clusterArn))
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime.IsZero() {
			err := tagEcsClusterWhenFirstSeen(awsSession, clusterArn, firstSeenTagKey, time.Now().UTC().Format(time.RFC3339))
			if err != nil {
				logging.Logger.Errorf("Error tagigng the ECS cluster with ARN %s", aws.StringValue(clusterArn))
				return nil, errors.WithStackTrace(err)
			}
		} else {
			if excludeAfter.After(firstSeenTime) {
				filteredEcsClusters = append(filteredEcsClusters, clusterArn)
			}
		}
	}

	return filteredEcsClusters, nil
}

func nukeEcsClusters(awsSession *session.Session, ecsClusterArns []*string) error {
	svc := ecs.New(awsSession)

	numNuking := len(ecsClusterArns)

	if numNuking == 0 {
		logging.Logger.Infof("No ECS clusters to nuke in region %s", aws.StringValue(awsSession.Config.Region))
		return nil
	}

	logging.Logger.Infof("Deleting %d ECS clusters in region %s", numNuking, aws.StringValue(awsSession.Config.Region))

	var nukedEcsClusters []*string
	for _, clusterArn := range ecsClusterArns {
		params := &ecs.DeleteClusterInput{
			Cluster: clusterArn,
		}
		_, err := svc.DeleteCluster(params)
		if err != nil {
			logging.Logger.Errorf("Error, failed to delete cluster with ARN %s", aws.StringValue(clusterArn))
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Success, deleted cluster: %s", aws.StringValue(clusterArn))
		nukedEcsClusters = append(nukedEcsClusters, clusterArn)
	}

	numNuked := len(nukedEcsClusters)
	logging.Logger.Infof("[OK] %d of %d ECS cluster(s) deleted in %s", numNuked, numNuking, aws.StringValue(awsSession.Config.Region))

	return nil
}

func getClusterTag(awsSession *session.Session, clusterArn *string, tagKey string) (time.Time, error) {
	var firstSeenTime time.Time

	svc := ecs.New(awsSession)
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := svc.ListTagsForResource(input)
	if err != nil {
		logging.Logger.Errorf("Error getting the tags for ECS cluster with ARN %s", aws.StringValue(clusterArn))
		return firstSeenTime, errors.WithStackTrace(err)
	}

	for _, tag := range clusterTags.Tags {
		if aws.StringValue(tag.Key) == tagKey {

			firstSeenTime, err := time.Parse(time.RFC3339, aws.StringValue(tag.Value))
			if err != nil {
				logging.Logger.Errorf("Error parsing the `cloud-nuke-first-seen` tag  into a `RFC3339` Time format for ECS cluster with ARN %s", aws.StringValue(clusterArn))
				return firstSeenTime, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}
	return firstSeenTime, nil
}
