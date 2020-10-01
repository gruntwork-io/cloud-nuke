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
func tagEcsCluster(awsSession *session.Session, clusterArn *string, tagKey string, tagValue string) (*ecs.TagResourceOutput, error) {
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

	result, err := svc.TagResource(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return result, nil
}

func getAllEcsClustersOlderThan(awsSession *session.Session, region string, t time.Time) ([]*string, error) {
	// get all ecs clusters
	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})

	clusterArns, err := getAllEcsClusters(awsSession)
	if err != nil {
		logging.Logger.Errorf("Error getting all ECS clusters")
		return nil, err
	}

	// var filteredEcsClusters []*string
	for _, clusterArn := range clusterArns {
		// if firstSeenTag is present
		// check if cluster is older than t
		// then include in filteredEcsClusters
		//	else
		// set first seen tag to time now

		firstSeen := getClusterTag(awsSession, clusterArn, "first_seen")

		if firstSeen == nil {
			_, err := tagEcsCluster(awsSession, clusterArn, "first_seen", "tomorrow")
			if err != nil {
			    return nil, err
			}
		}
	}
	return nil, nil
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
// }

func getClusterTag(awsSession *session.Session, clusterArn *string, tagKey string) *string {
	svc := ecs.New(awsSession)
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}

	clusterTags, err := svc.ListTagsForResource(input)
	if err != nil {
		logging.Logger.Errorf("Error getting the tags")
		return nil
	}

	for _, tag := range clusterTags.Tags {
		if aws.StringValue(tag.Key) == tagKey {
			//TODO turn from string into a timestamp
			return tag.Key
		}
	}

	return nil
}
