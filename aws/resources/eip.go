package resources

import (
	"context"
	goerr "errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of EIP allocation ids
func (eip *EIPAddresses) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	var firstSeenTime *time.Time
	var err error
	result, err := eip.Client.DescribeAddresses(eip.Context, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		firstSeenTime, err = util.GetOrCreateFirstSeen(c, eip.Client, address.AllocationId, util.ConvertTypesTagsToMap(address.Tags))
		if err != nil {
			logging.Error("unable to retrieve first seen tag")
			return nil, errors.WithStackTrace(err)
		}

		if eip.shouldInclude(address, firstSeenTime, configObj) {
			allocationIds = append(allocationIds, address.AllocationId)
		}
	}

	// checking the nukable permissions
	eip.VerifyNukablePermissions(allocationIds, func(id *string) error {
		_, err := eip.Client.ReleaseAddress(eip.Context, &ec2.ReleaseAddressInput{
			AllocationId: id,
			DryRun:       aws.Bool(true),
		})
		return err
	})

	return allocationIds, nil
}

func (eip *EIPAddresses) shouldInclude(address types.Address, firstSeenTime *time.Time, configObj config.Config) bool {
	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	allocationName := util.GetEC2ResourceNameTagValue(address.Tags)
	return configObj.ElasticIP.ShouldInclude(config.ResourceValue{
		Time: firstSeenTime,
		Name: allocationName,
		Tags: util.ConvertTypesTagsToMap(address.Tags),
	})
}

// Deletes all EIP allocation ids
func (eip *EIPAddresses) nukeAll(allocationIds []*string) error {
	if len(allocationIds) == 0 {
		logging.Debugf("No Elastic IPs to nuke in region %s", eip.Region)
		return nil
	}

	logging.Debugf("Deleting all Elastic IPs in region %s", eip.Region)
	var deletedAllocationIDs []*string

	for _, allocationID := range allocationIds {

		if nukable, reason := eip.IsNukable(aws.ToString(allocationID)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(allocationID), reason)
			continue
		}

		_, err := eip.Client.ReleaseAddress(eip.Context, &ec2.ReleaseAddressInput{
			AllocationId: allocationID,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(allocationID),
			ResourceType: "Elastic IP Address (EIP)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			var apiErr smithy.APIError
			if goerr.As(err, &apiErr) {
				switch apiErr.ErrorCode() {
				case "AuthFailure":
					// TODO: Figure out why we get an AuthFailure
					logging.Debugf("EIP %s can't be deleted, it is still attached to an active resource", *allocationID)
				default:
					logging.Debugf("[Failed] %s", err)
				}
			}
		} else {
			deletedAllocationIDs = append(deletedAllocationIDs, allocationID)
			logging.Debugf("Deleted Elastic IP: %s", *allocationID)
		}
	}

	logging.Debugf("[OK] %d Elastic IP(s) deleted in %s", len(deletedAllocationIDs), eip.Region)
	return nil
}
