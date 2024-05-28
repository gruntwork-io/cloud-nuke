package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of ASG Names
func (ag *ASGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := ag.Client.DescribeAutoScalingGroupsWithContext(ag.Context, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var groupNames []*string
	for _, group := range result.AutoScalingGroups {
		if configObj.AutoScalingGroup.ShouldInclude(config.ResourceValue{
			Time: group.CreatedTime,
			Name: group.AutoScalingGroupName,
			Tags: util.ConvertAutoScalingTagsToMap(group.Tags),
		}) {
			groupNames = append(groupNames, group.AutoScalingGroupName)
		}
	}

	return groupNames, nil
}

// Deletes all Auto Scaling Groups
func (ag *ASGroups) nukeAll(groupNames []*string) error {
	if len(groupNames) == 0 {
		logging.Debugf("No Auto Scaling Groups to nuke in region %s", ag.Region)
		return nil
	}

	logging.Debugf("Deleting all Auto Scaling Groups in region %s", ag.Region)
	var deletedGroupNames []*string

	for _, groupName := range groupNames {
		params := &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: groupName,
			ForceDelete:          awsgo.Bool(true),
		}

		_, err := ag.Client.DeleteAutoScalingGroupWithContext(ag.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   *groupName,
			ResourceType: "Auto-Scaling Group",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedGroupNames = append(deletedGroupNames, groupName)
			logging.Debugf("Deleted Auto Scaling Group: %s", *groupName)
		}
	}

	if len(deletedGroupNames) > 0 {
		err := ag.Client.WaitUntilGroupNotExistsWithContext(ag.Context, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: deletedGroupNames,
		})
		if err != nil {
			logging.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d Auto Scaling Group(s) deleted in %s", len(deletedGroupNames), ag.Region)
	return nil
}
