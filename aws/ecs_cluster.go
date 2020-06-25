package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// tagging an ECS cluster with a given tag key-value pair and a cluster ARN
func tagEcsCluster(awsSession *session.Session, clusterArn *string, tagKey string, tagValue string) (*ecs.TagResourceOutput, error) {
	logging.Logger.Infof("Tagging ECS cluster %s in region %s", clusterArn, *awsSession.Config.Region)

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

numNuking := len(ecsClusterArns)
	svc := ecs.New(awsSession)

	if numNuking == 0 {
		logging.Logger.Infof("No ECS clusters to nuke in region %s", *awsSession.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting %d ECS clusters in region %s", numNuking, *awsSession.Config.Region)

	numNuked := 0
	for _, ecsClusterArn := range ecsClusterArns {
		params := &ecs.DeleteClusterInput{Cluster: ecsClusterArn}
		_, err := svc.DeleteCluster(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] Could not delete cluster %s: %s", *ecsClusterArn, err.Error())
		} else {
			logging.Logger.Infof("Deleted cluster: %s", *ecsClusterArn)
			numNuked += 1
		}
	}
	logging.Logger.Infof("[OK] %d of %d ECS cluster(s) deleted in %s", numNuked, numNuking, *awsSession.Config.Region)
	return nil