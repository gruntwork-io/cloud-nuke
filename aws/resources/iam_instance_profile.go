package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// IAMInstanceProfilesAPI defines the interface for IAM instance profile operations.
type IAMInstanceProfilesAPI interface {
	ListInstanceProfiles(ctx context.Context, params *iam.ListInstanceProfilesInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesOutput, error)
	GetInstanceProfile(ctx context.Context, params *iam.GetInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.GetInstanceProfileOutput, error)
	RemoveRoleFromInstanceProfile(ctx context.Context, params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.RemoveRoleFromInstanceProfileOutput, error)
	DeleteInstanceProfile(ctx context.Context, params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteInstanceProfileOutput, error)
}

// NewIAMInstanceProfiles creates a new IAMInstanceProfiles resource using the generic resource pattern.
func NewIAMInstanceProfiles() AwsResource {
	return NewAwsResource(&resource.Resource[IAMInstanceProfilesAPI]{
		ResourceTypeName: "iam-instance-profile",
		BatchSize:        20,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[IAMInstanceProfilesAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.IAMInstanceProfiles
		},
		Lister: listIAMInstanceProfiles,
		// Instance profile deletion requires detaching roles first
		Nuker: resource.SequentialDeleter(deleteIAMInstanceProfile),
	})
}

// listIAMInstanceProfiles retrieves all IAM instance profiles that match the config filters.
func listIAMInstanceProfiles(ctx context.Context, client IAMInstanceProfilesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := iam.NewListInstanceProfilesPaginator(client, &iam.ListInstanceProfilesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list instance profiles: %v", err)
		}

		for _, profile := range output.InstanceProfiles {
			rv := config.ResourceValue{
				Name: profile.InstanceProfileName,
				Time: profile.CreateDate,
				Tags: map[string]string{},
			}
			for _, tag := range profile.Tags {
				if tag.Key == nil || tag.Value == nil {
					continue
				}
				rv.Tags[*tag.Key] = *tag.Value
			}
			if !cfg.ShouldInclude(rv) {
				continue
			}

			names = append(names, profile.InstanceProfileName)
		}
	}

	return names, nil
}

// deleteIAMInstanceProfile detaches all roles from an instance profile and then deletes it.
func deleteIAMInstanceProfile(ctx context.Context, client IAMInstanceProfilesAPI, profileName *string) error {
	// Get instance profile details
	profile, err := client.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: profileName,
	})
	if err != nil {
		return fmt.Errorf("failed to get instance profile %s: %w", aws.ToString(profileName), err)
	}

	// Detach roles from the instance profile
	for _, role := range profile.InstanceProfile.Roles {
		_, err := client.RemoveRoleFromInstanceProfile(ctx, &iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: profileName,
			RoleName:            role.RoleName,
		})
		if err != nil {
			return fmt.Errorf("failed to remove role %s from instance profile %s: %w", aws.ToString(role.RoleName), aws.ToString(profileName), err)
		}
		logging.Debugf("Detached role %s from instance profile %s", aws.ToString(role.RoleName), aws.ToString(profileName))
	}

	// Delete the instance profile
	_, err = client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
		InstanceProfileName: profileName,
	})
	if err != nil {
		return fmt.Errorf("failed to delete instance profile %s: %w", aws.ToString(profileName), err)
	}
	logging.Debugf("Successfully deleted instance profile: %s", aws.ToString(profileName))

	return nil
}
