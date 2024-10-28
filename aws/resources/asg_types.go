package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ASGroupsAPI interface {
	DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
	DeleteAutoScalingGroup(ctx context.Context, params *autoscaling.DeleteAutoScalingGroupInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteAutoScalingGroupOutput, error)
}

// ASGroups - represents all auto-scaling groups
type ASGroups struct {
	BaseAwsResource
	Client     ASGroupsAPI
	Region     string
	GroupNames []string
}

func (ag *ASGroups) InitV2(cfg aws.Config) {
	ag.Client = autoscaling.NewFromConfig(cfg)
}

func (ag *ASGroups) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (ag *ASGroups) ResourceName() string {
	return "asg"
}

func (ag *ASGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The group names of the auto-scaling groups
func (ag *ASGroups) ResourceIdentifiers() []string {
	return ag.GroupNames
}

func (ag *ASGroups) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.AutoScalingGroup
}
func (ag *ASGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ag.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ag.GroupNames = aws.ToStringSlice(identifiers)
	return ag.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (ag *ASGroups) Nuke(identifiers []string) error {
	if err := ag.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
