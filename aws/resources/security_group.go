package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	r "github.com/gruntwork-io/cloud-nuke/report" // Alias the package as 'r'
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (sg *SecurityGroup) setFirstSeenTag(securityGroup ec2.SecurityGroup, value time.Time) error {
	_, err := sg.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{securityGroup.GroupId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(util.FirstSeenTagKey),
				Value: aws.String(util.FormatTimestamp(value)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func (sg *SecurityGroup) getFirstSeenTag(securityGroup ec2.SecurityGroup) (*time.Time, error) {
	tags := securityGroup.Tags
	for _, tag := range tags {
		if util.IsFirstSeenTag(tag.Key) {
			firstSeenTime, err := util.ParseTimestamp(tag.Value)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return firstSeenTime, nil
		}
	}
	return nil, nil
}

// shouldIncludeSecurityGroup determines whether a security group should be included for deletion based on the provided configuration.
func shouldIncludeSecurityGroup(sg *ec2.SecurityGroup, firstSeenTime *time.Time, configObj config.Config) bool {
	var groupName = sg.GroupName
	return configObj.SecurityGroup.ShouldInclude(config.ResourceValue{
		Name: groupName,
		Tags: util.ConvertEC2TagsToMap(sg.Tags),
		Time: firstSeenTime,
	})
}

// getAll retrieves all security group identifiers based on the provided configuration.
func (sg *SecurityGroup) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	resp, err := sg.Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		logging.Debugf("[Security Group] Failed to list security groups: %s", err)
		return nil, errors.WithStackTrace(err)
	}

	for _, group := range resp.SecurityGroups {
		// check first seen tag
		firstSeenTime, err := sg.getFirstSeenTag(*group)
		if err != nil {
			logging.Errorf(
				"Unable to retrieve tags for Security group: %s, with error: %s", *group.GroupId, err)
			continue
		}

		// if the first seen tag is not there, then create one
		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := sg.setFirstSeenTag(*group, time.Now().UTC()); err != nil {
				logging.Errorf(
					"Unable to apply first seen tag Security group: %s, with error: %s", *group.GroupId, err)
				continue
			}
		}
		if shouldIncludeSecurityGroup(group, firstSeenTime, configObj) {
			identifiers = append(identifiers, group.GroupId)
		}
	}

	// Check and verify the list of allowed nuke actions
	sg.VerifyNukablePermissions(identifiers, func(id *string) error {
		_, err := sg.Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: id,
			DryRun:  aws.Bool(true),
		})
		return err
	})

	return identifiers, nil
}

func (sg *SecurityGroup) nuke(id *string) error {

	if err := sg.terminateInstancesAssociatedWithSecurityGroup(*id); err != nil {
		return errors.WithStackTrace(err)
	}

	if err := sg.nukeSecurityGroup(*id); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func (sg *SecurityGroup) terminateInstancesAssociatedWithSecurityGroup(id string) error {

	resp, err := sg.Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []*string{aws.String(id)},
			},
		},
	})
	if err != nil {
		logging.Debugf("[Security Group] Failed to describe instances associated with security group %s: %s", id, err)
		return errors.WithStackTrace(err)
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := aws.StringValue(instance.InstanceId)

			// Needs to release the elastic ips attached on the instance before nuking
			if err := sg.releaseEIPs([]*string{instance.InstanceId}); err != nil {
				logging.Debugf("[Failed EIP release] %s", err)
				return errors.WithStackTrace(err)
			}

			// terminating the instances which used this security group
			if _, err := sg.Client.TerminateInstances(&ec2.TerminateInstancesInput{
				InstanceIds: []*string{aws.String(instanceID)},
			}); err != nil {
				logging.Debugf("[Failed] Ec2 termination %s", err)
				return errors.WithStackTrace(err)
			}

			logging.Debugf("[Instance Termination] waiting to terminate instance %s", instanceID)

			// wait until the instance terminated.
			if err := sg.Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
				InstanceIds: []*string{aws.String(instanceID)},
			}); err != nil {
				logging.Debugf("[Security Group] Failed to terminate instance %s associated with security group %s: %s", instanceID, id, err)
				return errors.WithStackTrace(err)
			}

			logging.Debugf("Terminated instance %s associated with security group %s", instanceID, id)
		}
	}

	return nil
}

func (sg *SecurityGroup) releaseEIPs(instanceIds []*string) error {
	logging.Debugf("Releasing Elastic IP address(s) associated on instances")
	for _, instanceID := range instanceIds {

		// get the elastic ip's associated with the EC2's
		output, err := sg.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("instance-id"),
					Values: []*string{
						instanceID,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		for _, address := range output.Addresses {
			if _, err := sg.Client.ReleaseAddress(&ec2.ReleaseAddressInput{
				AllocationId: address.AllocationId,
			}); err != nil {
				logging.Debugf("An error happened while releasing the elastic ip address %s, error %v", *address.AllocationId, err)
				continue
			}

			logging.Debugf("Released Elastic IP address %s from instance %s", *address.AllocationId, *instanceID)
		}
	}

	return nil
}

func (sg *SecurityGroup) nukeSecurityGroup(id string) error {
	if _, err := sg.Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(id),
	}); err != nil {
		logging.Debugf("[Security Group] Failed to delete security group %s: %s", id, err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("Deleted security group %s", id)

	return nil
}

func (sg *SecurityGroup) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No security group identifiers to nuke in region %s", sg.Region)
		return nil
	}

	logging.Debugf("Deleting all security groups in region %s", sg.Region)
	var deletedGroups []*string

	for _, id := range identifiers {

		if nukable, err := sg.IsNukable(*id); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		err := sg.nuke(id)
		// Record status of this resource
		e := r.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Security Group",
			Error:        err,
		}
		r.Record(e)

		if err != nil {
			logging.Debugf("[Failed] Error deleting security group %s: %s", *id, err)
		} else {
			deletedGroups = append(deletedGroups, id)
			logging.Debugf("Deleted security group: %s", *id)
		}
	}

	logging.Debugf("[OK] %d security group(s) deleted in %s", len(deletedGroups), sg.Region)

	return nil
}
