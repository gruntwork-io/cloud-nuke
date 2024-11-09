package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

// getAll extracts the list of existing ec2 placement groups
func (k *EC2PlacementGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var names []*string
	var firstSeenTime *time.Time

	result, err := k.Client.DescribePlacementGroups(k.Context, &ec2.DescribePlacementGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	for _, placementGroup := range result.PlacementGroups {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, k.Client, placementGroup.GroupId, util.ConvertTypesTagsToMap(placementGroup.Tags))
		if err != nil {
			logging.Error("Unable to retrieve tags")
			return nil, errors.WithStackTrace(err)
		}

		if configObj.EC2PlacementGroups.ShouldInclude(config.ResourceValue{
			Name: placementGroup.GroupName,
			Time: firstSeenTime,
			Tags: util.ConvertTypesTagsToMap(placementGroup.Tags),
		}) {
			names = append(names, placementGroup.GroupName)
		}
	}

	// checking the nukable permissions
	k.VerifyNukablePermissions(names, func(name *string) error {
		_, err := k.Client.DeletePlacementGroup(k.Context, &ec2.DeletePlacementGroupInput{
			GroupName: name,
			DryRun:    aws.Bool(true),
		})
		return err
	})

	return names, nil
}

// deleteKeyPair is a helper method that deletes the given ec2 key pair.
func (k *EC2PlacementGroups) deletePlacementGroup(placementGroupName *string) error {
	params := &ec2.DeletePlacementGroupInput{
		GroupName: placementGroupName,
	}

	_, err := k.Client.DeletePlacementGroup(k.Context, params)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// nukeAllEc2KeyPairs attempts to delete given ec2 key pair IDs.
func (k *EC2PlacementGroups) nukeAll(groupNames []*string) error {
	if len(groupNames) == 0 {
		logging.Infof("No EC2 placement groups to nuke in region %s", k.Region)
		return nil
	}

	logging.Infof("Terminating all EC2 placement groups in region %s", k.Region)

	deletedPlacementGroups := 0
	var multiErr *multierror.Error
	for _, groupName := range groupNames {
		if nukable, reason := k.IsNukable(aws.ToString(groupName)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(groupName), reason)
			continue
		}

		if err := k.deletePlacementGroup(groupName); err != nil {
			logging.Errorf("[Failed] %s", err)
			multiErr = multierror.Append(multiErr, err)
		} else {
			deletedPlacementGroups++
			logging.Infof("Deleted EC2 Placement Group: %s", *groupName)
		}
	}

	logging.Infof("[OK] %d EC2 Placement Group(s) terminated", deletedPlacementGroups)
	return multiErr.ErrorOrNil()
}
