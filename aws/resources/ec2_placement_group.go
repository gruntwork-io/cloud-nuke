package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// EC2PlacementGroupsAPI defines the interface for EC2 Placement Groups operations.
type EC2PlacementGroupsAPI interface {
	DescribePlacementGroups(ctx context.Context, params *ec2.DescribePlacementGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribePlacementGroupsOutput, error)
	DeletePlacementGroup(ctx context.Context, params *ec2.DeletePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeletePlacementGroupOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewEC2PlacementGroups creates a new EC2 Placement Groups resource using the generic resource pattern.
func NewEC2PlacementGroups() AwsResource {
	return NewAwsResource(&resource.Resource[EC2PlacementGroupsAPI]{
		ResourceTypeName: "ec2-placement-groups",
		BatchSize:        200,
		InitClient: func(r *resource.Resource[EC2PlacementGroupsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2PlacementGroups
		},
		Lister:             listEC2PlacementGroups,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2PlacementGroup),
		PermissionVerifier: verifyEC2PlacementGroupPermission,
	})
}

// listEC2PlacementGroups retrieves all EC2 placement groups that match the config filters.
func listEC2PlacementGroups(ctx context.Context, client EC2PlacementGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribePlacementGroups(ctx, &ec2.DescribePlacementGroupsInput{})
	if err != nil {
		return nil, err
	}

	var names []*string
	for _, placementGroup := range result.PlacementGroups {
		firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, placementGroup.GroupId, util.ConvertTypesTagsToMap(placementGroup.Tags))
		if err != nil {
			logging.Errorf("Unable to retrieve tags for placement group %s: %v", aws.ToString(placementGroup.GroupName), err)
			return nil, err
		}

		if cfg.ShouldInclude(config.ResourceValue{
			Name: placementGroup.GroupName,
			Time: firstSeenTime,
			Tags: util.ConvertTypesTagsToMap(placementGroup.Tags),
		}) {
			names = append(names, placementGroup.GroupName)
		}
	}

	return names, nil
}

// deleteEC2PlacementGroup deletes a single EC2 placement group.
func deleteEC2PlacementGroup(ctx context.Context, client EC2PlacementGroupsAPI, name *string) error {
	_, err := client.DeletePlacementGroup(ctx, &ec2.DeletePlacementGroupInput{
		GroupName: name,
	})
	return err
}

// verifyEC2PlacementGroupPermission performs a dry-run delete to check permissions.
func verifyEC2PlacementGroupPermission(ctx context.Context, client EC2PlacementGroupsAPI, name *string) error {
	_, err := client.DeletePlacementGroup(ctx, &ec2.DeletePlacementGroupInput{
		GroupName: name,
		DryRun:    aws.Bool(true),
	})
	return err
}
