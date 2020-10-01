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

	//not interested in the output for now
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
		// if firstSeenTag is present
		// check if cluster is older than t
		// then include in filteredEcsClusters
		//	else
		// set first seen tag to time now

		firstSeenTime, err := getClusterTag(awsSession, clusterArn, "first_seen")
		if err != nil {
			logging.Logger.Errorf("Error getting the `first_seen` ECS cluster") //todo - add specific cluster Arn
			return nil, err
		}

		if firstSeenTime == nil {
			err := tagEcsCluster(awsSession, clusterArn, "first_seen", time.Now().UTC().String())
			if err != nil {
				logging.Logger.Errorf("Error tagigng the ECS cluster") //todo - add specific cluster Arn
				return nil, err
			}
		}

		// todo - filter based on "older than"
		if firstSeenTime.After(timeMargin) {
			filteredEcsClusters = append(filteredEcsClusters, clusterArn)
		}
	}

	return filteredEcsClusters, nil
}

// func nukeEcsClusters(awsSession *session.Session, ecsClusterArns []*string) error {
// 	svc := ecs.New(awsSession)
// 	for _, clusterArn := range ecsClusterArns {

// 		if tag is present then delete
// 		if tagIsPresent(awsSession, clusterArn) == true {
// 			params := &ecs.DeleteClusterInput{Cluster: clusterArn}
// 			_, err := svc.DeleteCluster(params)
// 			if err != nil {
// 				logging.Logger.Errorf("Error, failed to delete cluster")
// 			} else {
// 				logging.Logger.Infof("Success, deleted cluster: %s", *clusterArn)
// 			}
// 		}
// 		if tag is not present set the "first_seen" tag
// 	}

// 	//to do - return a list of the deleted clusters (name)
// 	return error
// }(

func getClusterTag(awsSession *session.Session, clusterArn *string, tagKey string) (*time.Time, error) {
	svc := ecs.New(awsSession)
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := svc.ListTagsForResource(input)
	if err != nil {
		logging.Logger.Errorf("Error getting the tags")
		return nil, nil
	}

	// const key = "cloud-nuke-first-seen"
	const layout = "2006-01-02 15:04:05"
	var firstSeenTime *time.Time

	for _, tag := range clusterTags.Tags {
		if aws.StringValue(tag.Key) == tagKey {
			firstSeenTime, err := time.Parse(layout, *tag.Value)
			if err != nil {
				logging.Logger.Errorf("Error tagigng the ECS cluster") //todo - add specific cluster Arn
				return nil, errors.WithStackTrace(err)
			}
			return firstSeenTime, nil
		}
	}
	return nil, nil
}
