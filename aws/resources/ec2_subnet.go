package resources

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (ec2subnet *EC2Subnet) setFirstSeenTag(sb ec2.Subnet, value time.Time) error {
	_, err := ec2subnet.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{sb.SubnetId},
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

func (ec2subnet *EC2Subnet) getFirstSeenTag(sb ec2.Subnet) (*time.Time, error) {
	tags := sb.Tags
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

func shouldIncludeEc2Subnet(subnet *ec2.Subnet, firstSeenTime *time.Time, configObj config.Config) bool {
	var subnetName string
	tagMap := util.ConvertEC2TagsToMap(subnet.Tags)
	if name, ok := tagMap["Name"]; ok {
		subnetName = name
	}

	return configObj.EC2Subnet.ShouldInclude(config.ResourceValue{
		Name: &subnetName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// Returns a formatted string of EC2 subnets
func (ec2subnet *EC2Subnet) getAll(_ context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}
	// Note: This filter initially handles non-default resources and can be overridden by passing the only-default filter to choose default subnets.
	if configObj.EC2Subnet.DefaultOnly {
		logging.Debugf("[default only] Retrieving the default subnets")
	}

	err := ec2subnet.Client.DescribeSubnetsPages(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("default-for-az"),
				Values: []*string{
					aws.String(strconv.FormatBool(configObj.EC2Subnet.DefaultOnly)), // convert the bool status into string
				},
			},
		},
	}, func(pages *ec2.DescribeSubnetsOutput, lastPage bool) bool {
		for _, subnet := range pages.Subnets {

			// check first seen tag
			firstSeenTime, err := ec2subnet.getFirstSeenTag(*subnet)
			if err != nil {
				logging.Errorf(
					"Unable to retrieve tags for Subnet: %s, with error: %s", *subnet.SubnetId, err)
				continue
			}

			// if the first seen tag is not there, then create one
			if firstSeenTime == nil {
				now := time.Now().UTC()
				firstSeenTime = &now
				if err := ec2subnet.setFirstSeenTag(*subnet, time.Now().UTC()); err != nil {
					logging.Errorf(
						"Unable to apply first seen tag Subnet: %s, with error: %s", *subnet.SubnetId, err)
					continue
				}
			}

			if shouldIncludeEc2Subnet(subnet, firstSeenTime, configObj) {
				result = append(result, subnet.SubnetId)
			}
		}
		return !lastPage
	})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// check the resources are nukable
	ec2subnet.VerifyNukablePermissions(result, func(id *string) error {
		params := &ec2.DeleteSubnetInput{
			SubnetId: id,
			DryRun:   aws.Bool(true), // dry run set as true , checks permission without actually making the request
		}
		_, err := ec2subnet.Client.DeleteSubnet(params)
		return err
	})

	return result, nil
}

// Deletes all Subnets
func (ec2subnet *EC2Subnet) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Subnets to nuke in region %s", ec2subnet.Region)
		return nil
	}

	logging.Debugf("Deleting all Subnets in region %s", ec2subnet.Region)
	var deletedAddresses []*string

	for _, id := range ids {
		// check the id has the permission to nuke, if not. continue the execution
		if nukable, err := ec2subnet.IsNukable(*id); !nukable {
			// not adding the report on final result hence not adding a record entry here
			// NOTE: We can skip the error checking and return it here, since it is already being checked while
			// displaying the identifiers. Here, `err` refers to the error indicating whether the identifier is eligible for nuke or not,
			// and it is not a programming error.
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		_, err := ec2subnet.Client.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Subnet",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedAddresses = append(deletedAddresses, id)
			logging.Debugf("Deleted Subnet: %s", *id)
		}
	}

	logging.Debugf("[OK] %d EC2 Subnet(s) deleted in %s", len(deletedAddresses), ec2subnet.Region)

	return nil
}
