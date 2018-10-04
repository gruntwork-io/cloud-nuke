package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of ASG Names
func getAllAutoScalingGroups(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := autoscaling.New(session)
	result, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var groupNames []*string
	for _, group := range result.AutoScalingGroups {
		if excludeAfter.After(*group.CreatedTime) {
			groupNames = append(groupNames, group.AutoScalingGroupName)
		}
	}

	return groupNames, nil
}

// Deletes all Auto Scaling Groups
func nukeAllAutoScalingGroups(session *session.Session, groupNames []*string) error {
	svc := autoscaling.New(session)

	if len(groupNames) == 0 {
		logging.Logger.Infof("No Auto Scaling Groups to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Auto Scaling Groups in region %s", *session.Config.Region)
	var deletedGroupNames []*string

	for _, groupName := range groupNames {
		params := &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: groupName,
			ForceDelete:          awsgo.Bool(true),
		}

		_, err := svc.DeleteAutoScalingGroup(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedGroupNames = append(deletedGroupNames, groupName)
			logging.Logger.Infof("Deleted Auto Scaling Group: %s", *groupName)
		}
	}

	err := svc.WaitUntilGroupNotExists(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: deletedGroupNames,
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Logger.Infof("[OK] %d Auto Scaling Group(s) deleted in %s", len(deletedGroupNames), *session.Config.Region)
	return nil
}
