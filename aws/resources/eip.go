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

// Returns a formatted string of EIP allocation ids
func (ea *EIPAddresses) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	var firstSeenTime *time.Time
	var err error
	result, err := ea.Client.DescribeAddressesWithContext(ea.Context, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, ea.Client, address.AllocationId, util.ConvertEC2TagsToMap(address.Tags))
		if err != nil {
			logging.Error("unable to retrieve first seen tag")
			return nil, errors.WithStackTrace(err)
		}

		if ea.shouldInclude(address, firstSeenTime, configObj) {
			allocationIds = append(allocationIds, address.AllocationId)
		}
	}

	// checking the nukable permissions
	ea.VerifyNukablePermissions(allocationIds, func(id *string) error {
		_, err := ea.Client.ReleaseAddressWithContext(ea.Context, &ec2.ReleaseAddressInput{
			AllocationId: id,
			DryRun:       awsgo.Bool(true),
		})
		return err
	})

	return allocationIds, nil
}

func (ea *EIPAddresses) shouldInclude(address *ec2.Address, firstSeenTime *time.Time, configObj config.Config) bool {
	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	allocationName := util.GetEC2ResourceNameTagValue(address.Tags)
	return configObj.ElasticIP.ShouldInclude(config.ResourceValue{
		Time: firstSeenTime,
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

		if nukable, reason := ea.IsNukable(awsgo.StringValue(allocationID)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(allocationID), reason)
			continue
		}

		_, err := ea.Client.ReleaseAddressWithContext(ea.Context, &ec2.ReleaseAddressInput{
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
