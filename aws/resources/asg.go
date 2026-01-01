package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
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
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ASGroupsAPI], cfg aws.Config) {
			r.Client = autoscaling.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AutoScalingGroup
		},
		Lister: listASGroups,
		Nuker:  resource.SequentialDeleteThenWaitAll(deleteASG, waitForASGsDeleted),
	})
}

// listASGroups retrieves all Auto Scaling Groups that match the config filters.
func listASGroups(ctx context.Context, client ASGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var groupNames []*string
	paginator := autoscaling.NewDescribeAutoScalingGroupsPaginator(client, &autoscaling.DescribeAutoScalingGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, group := range page.AutoScalingGroups {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: group.CreatedTime,
				Name: group.AutoScalingGroupName,
				Tags: util.ConvertAutoScalingTagsToMap(group.Tags),
			}) {
				groupNames = append(groupNames, group.AutoScalingGroupName)
			}
		}
	}

	return groupNames, nil
}

// deleteASG deletes a single Auto Scaling Group by name.
func deleteASG(ctx context.Context, client ASGroupsAPI, name *string) error {
	_, err := client.DeleteAutoScalingGroup(ctx, &autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: name,
		ForceDelete:          aws.Bool(true),
	})
	return errors.WithStackTrace(err)
}

// waitForASGsDeleted waits for all specified Auto Scaling Groups to be deleted.
func waitForASGsDeleted(ctx context.Context, client ASGroupsAPI, names []string) error {
	waiter := autoscaling.NewGroupNotExistsWaiter(client)
	return waiter.Wait(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: names,
	}, 5*time.Minute)
}
