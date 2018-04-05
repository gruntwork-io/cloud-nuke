package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of EIP allocation ids
func getAllEIPAddresses(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)

	result, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		allocationIds = append(allocationIds, address.AllocationId)
	}

	return allocationIds, nil
}

// Deletes all EIP allocation ids
func nukeAllEIPAddresses(session *session.Session, allocationIds []*string) error {
	svc := ec2.New(session)

	if len(allocationIds) == 0 {
		logging.Logger.Infof("No Elastic IPs to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Elastic IPs in region %s", *session.Config.Region)

	for _, allocationID := range allocationIds {
		params := &ec2.ReleaseAddressInput{
			AllocationId: allocationID,
		}

		_, err := svc.ReleaseAddress(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted Elastic IP: %s", *allocationID)
	}

	logging.Logger.Infof("[OK] %d Elastc IP(s) deleted in %s", len(allocationIds), *session.Config.Region)
	return nil
}
