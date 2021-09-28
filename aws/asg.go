package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
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
		if excludeAfter.After(*group.CreatedTime) && !hasASGExcludeTag(group) {
			groupNames = append(groupNames, group.AutoScalingGroupName)
		} else if !hasASGExcludeTag(group) {
			groupNames = append(groupNames, group.AutoScalingGroupName)
		}
	}

	return groupNames, nil
}

// hasASGExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasASGExcludeTag(group *autoscaling.Group) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	for _, tag := range group.Tags {
		if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
			return true
		}
	}
	return false
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

	if len(deletedGroupNames) > 0 {
		err := svc.WaitUntilGroupNotExists(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: deletedGroupNames,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Infof("[OK] %d Auto Scaling Group(s) deleted in %s", len(deletedGroupNames), *session.Config.Region)
	return nil
}
