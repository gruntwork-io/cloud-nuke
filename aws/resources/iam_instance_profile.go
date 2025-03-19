package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
)

// Returns the Names of all iam instance profiles
func (ip *IAMInstanceProfiles) getAll(c context.Context, configObj config.Config) (names []*string, err error) {
	paginator := iam.NewListInstanceProfilesPaginator(ip.Client, &iam.ListInstanceProfilesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(c)
		if err != nil {
			return nil, fmt.Errorf("failed to list instance profiles: %v", err)
		}

		for _, profile := range output.InstanceProfiles {
			rv := config.ResourceValue{
				Name: profile.InstanceProfileName,
				Time: profile.CreateDate,
				Tags: map[string]string{},
			}
			for _, tags := range profile.Tags {
				if tags.Key == nil || tags.Value == nil {
					continue
				}
				rv.Tags[*tags.Key] = *tags.Value
			}
			if !configObj.IAMInstanceProfiles.ShouldInclude(rv) {
				continue
			}

			names = append(names, profile.InstanceProfileName)
		}
	}

	return
}

// Delete all iam instance profiles.
func (ip *IAMInstanceProfiles) nukeAll(names []*string) (err error) {
	if len(names) == 0 {
		logging.Debug("No IAM Instance Profiles to nuke")
		return
	}

	for _, name := range names {
		ctx := context.Background()

		// Get instance profile details
		profile, err := ip.Client.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
			InstanceProfileName: name,
		})
		if err != nil {
			return fmt.Errorf("failed to get instance profile %s: %v", *name, err)
		}

		// Detach roles from the instance profile
		for _, role := range profile.InstanceProfile.Roles {
			_, err := ip.Client.RemoveRoleFromInstanceProfile(ctx, &iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: name,
				RoleName:            role.RoleName,
			})
			if err != nil {
				return fmt.Errorf("failed to remove role %s from instance profile %s: %v", *role.RoleName, *name, err)
			}
			logging.Debugf("Detached role %s from from instance profile %s \n", *role.RoleName, *name)
		}

		// Delete the instance profile
		_, err = ip.Client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
			InstanceProfileName: name,
		})
		if err != nil {
			return fmt.Errorf("failed to delete instance profile: %v", err)
		}
		logging.Debugf("Successfully deleted instance profile: %s", *name)

	}

	return
}
