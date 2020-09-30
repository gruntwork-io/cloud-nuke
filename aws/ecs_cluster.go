package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// tagging an ECS cluster with a given tag key-value pair and a cluster ARN
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

func nukeEcsClusterByTag(awsSession *session.Session, ecsClusterArns []*string) error {
	svc := ecs.New(awsSession)
	for _, clusterArn := range ecsClusterArns {

		//if tag is present then delete
		if tagIsPresent(awsSession, clusterArn) == true {
			params := &ecs.DeleteClusterInput{Cluster: clusterArn}
			_, err := svc.DeleteCluster(params)
			if err != nil {
				logging.Logger.Errorf("Error, failed to delete cluster")
			} else {
				logging.Logger.Infof("Success, deleted cluster: %s", *clusterArn)
			}
		}
	}

	return nil
}

func tagIsPresent(awsSession *session.Session, clusterArn *string) bool {
	svc := ecs.New(awsSession)
	input := &ecs.ListTagsForResourceInput{
		ResourceArn: clusterArn,
	}
	result, err := svc.ListTagsForResource(input)
	if err != nil {
		logging.Logger.Errorf("error getting the tags")
		return false
	}
	fmt.Println(result)
	return true
}
