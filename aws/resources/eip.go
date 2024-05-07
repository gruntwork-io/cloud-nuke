package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (ea *EIPAddresses) setFirstSeenTag(address ec2.Address, value time.Time) error {
	// We set a first seen tag because an Elastic IP doesn't contain an attribute that gives us it's creation time
	_, err := ea.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{address.AllocationId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(util.FirstSeenTagKey),
				Value: awsgo.String(util.FormatTimestamp(value)),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (ea *EIPAddresses) getFirstSeenTag(address ec2.Address) (*time.Time, error) {
	tags := address.Tags
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

// Returns a formatted string of EIP allocation ids
func (ea *EIPAddresses) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := ea.Client.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		firstSeenTime, err := ea.getFirstSeenTag(*address)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := ea.setFirstSeenTag(*address, *firstSeenTime); err != nil {
				return nil, err
			}
		}
		if ea.shouldInclude(address, *firstSeenTime, configObj) {
			allocationIds = append(allocationIds, address.AllocationId)
		}
	}

	// checking the nukable permissions
	ea.VerifyNukablePermissions(allocationIds, func(id *string) error {
		_, err := ea.Client.ReleaseAddress(&ec2.ReleaseAddressInput{
			AllocationId: id,
			DryRun:       awsgo.Bool(true),
		})
		return err
	})

	return allocationIds, nil
}

func (ea *EIPAddresses) shouldInclude(address *ec2.Address, firstSeenTime time.Time, configObj config.Config) bool {
	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	allocationName := util.GetEC2ResourceNameTagValue(address.Tags)
	return configObj.ElasticIP.ShouldInclude(config.ResourceValue{
		Time: &firstSeenTime,
		Name: allocationName,
		Tags: util.ConvertEC2TagsToMap(address.Tags),
	})
}

// Deletes all EIP allocation ids
func (ea *EIPAddresses) nukeAll(allocationIds []*string) error {
	if len(allocationIds) == 0 {
		logging.Debugf("No Elastic IPs to nuke in region %s", ea.Region)
		return nil
	}

	logging.Debugf("Deleting all Elastic IPs in region %s", ea.Region)
	var deletedAllocationIDs []*string

	for _, allocationID := range allocationIds {

		if nukable, err := ea.IsNukable(awsgo.StringValue(allocationID)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(allocationID), err)
			continue
		}

		_, err := ea.Client.ReleaseAddress(&ec2.ReleaseAddressInput{
			AllocationId: allocationID,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(allocationID),
			ResourceType: "Elastic IP Address (EIP)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "AuthFailure" {
				// TODO: Figure out why we get an AuthFailure
				logging.Debugf("EIP %s can't be deleted, it is still attached to an active resource", *allocationID)
			} else {
				logging.Debugf("[Failed] %s", err)
			}
		} else {
			deletedAllocationIDs = append(deletedAllocationIDs, allocationID)
			logging.Debugf("Deleted Elastic IP: %s", *allocationID)
		}
	}

	logging.Debugf("[OK] %d Elastic IP(s) deleted in %s", len(deletedAllocationIDs), ea.Region)
	return nil
}
