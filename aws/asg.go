package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of ASG Names
func getAllAutoScalingGroups(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := autoscaling.New(session)
	result, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var groupNames []*string
	for _, group := range result.AutoScalingGroups {
		if shouldIncludeAutoScalingGroup(group, excludeAfter, configObj) {
			groupNames = append(groupNames, group.AutoScalingGroupName)
		}
	}

	return groupNames, nil
}

func shouldIncludeAutoScalingGroup(group *autoscaling.Group, excludeAfter time.Time, configObj config.Config) bool {
	if group == nil {
		return false
	}

	if group.CreatedTime != nil && excludeAfter.Before(*group.CreatedTime) {
		return false
	}

	return config.ShouldInclude(
		awsgo.StringValue(group.AutoScalingGroupName),
		configObj.AutoScalingGroup.IncludeRule.NamesRegExp,
		configObj.AutoScalingGroup.ExcludeRule.NamesRegExp,
	)
}

// Deletes all Auto Scaling Groups
func nukeAllAutoScalingGroups(session *session.Session, groupNames []*string) error {
	svc := autoscaling.New(session)

	if len(groupNames) == 0 {
		logging.Logger.Debugf("No Auto Scaling Groups to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Auto Scaling Groups in region %s", *session.Config.Region)
	var deletedGroupNames []*string

	for _, groupName := range groupNames {
		params := &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: groupName,
			ForceDelete:          awsgo.Bool(true),
		}

		_, err := svc.DeleteAutoScalingGroup(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   *groupName,
			ResourceType: "Auto-Scaling Group",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedGroupNames = append(deletedGroupNames, groupName)
			logging.Logger.Debugf("Deleted Auto Scaling Group: %s", *groupName)
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

	logging.Logger.Debugf("[OK] %d Auto Scaling Group(s) deleted in %s", len(deletedGroupNames), *session.Config.Region)
	return nil
}
