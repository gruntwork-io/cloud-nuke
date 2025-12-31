package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// ASGroupsAPI defines the interface for Auto Scaling Group operations.
type ASGroupsAPI interface {
	DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
	DeleteAutoScalingGroup(ctx context.Context, params *autoscaling.DeleteAutoScalingGroupInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteAutoScalingGroupOutput, error)
}

// NewASGroups creates a new ASGroups resource using the generic resource pattern.
func NewASGroups() AwsResource {
	return NewAwsResource(&resource.Resource[ASGroupsAPI]{
		ResourceTypeName: "asg",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[ASGroupsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for ASGroups client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = autoscaling.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AutoScalingGroup
		},
		Lister: listASGroups,
		Nuker:  deleteASGroups,
	})
}

// listASGroups retrieves all Auto Scaling Groups that match the config filters.
func listASGroups(ctx context.Context, client ASGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var groupNames []*string
	for _, group := range result.AutoScalingGroups {
		if cfg.ShouldInclude(config.ResourceValue{
			Time: group.CreatedTime,
			Name: group.AutoScalingGroupName,
			Tags: util.ConvertAutoScalingTagsToMap(group.Tags),
		}) {
			groupNames = append(groupNames, group.AutoScalingGroupName)
		}
	}

	return groupNames, nil
}

// deleteASGroups deletes all Auto Scaling Groups.
func deleteASGroups(ctx context.Context, client ASGroupsAPI, scope resource.Scope, resourceType string, groupNames []*string) error {
	if len(groupNames) == 0 {
		logging.Debugf("No Auto Scaling Groups to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all Auto Scaling Groups in region %s", scope.Region)
	var deletedGroupNames []string

	for _, groupName := range groupNames {
		params := &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: groupName,
			ForceDelete:          aws.Bool(true),
		}

		_, err := client.DeleteAutoScalingGroup(ctx, params)

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
			deletedGroupNames = append(deletedGroupNames, *groupName)
			logging.Debugf("Deleted Auto Scaling Group: %s", *groupName)
		}
	}

	if len(deletedGroupNames) > 0 {
		waiter := autoscaling.NewGroupNotExistsWaiter(client)
		err := waiter.Wait(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: deletedGroupNames,
		}, 5*time.Minute)

		if err != nil {
			logging.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d Auto Scaling Group(s) deleted in %s", len(deletedGroupNames), scope.Region)
	return nil
}
