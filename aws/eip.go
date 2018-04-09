package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func setFirstSeenTag(svc *ec2.EC2, address ec2.Address, key string, value time.Time, layout string) error {
	// We set a first seen tag because an Elastic IP doesn't contain an attribute that gives us it's creation time
	_, err := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{address.AllocationId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String(key),
				Value: awsgo.String(value.Format(layout)),
			},
		},
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func getFirstSeenTag(svc *ec2.EC2, address ec2.Address, key string, layout string) (*time.Time, error) {
	tags := address.Tags
	for _, tag := range tags {
		if *tag.Key == key {
			firstSeenTime, err := time.Parse(layout, *tag.Value)
			return &firstSeenTime, err
		}
	}

	return nil, nil
}

// Returns a formatted string of EIP allocation ids
func getAllEIPAddresses(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)
	const key = "cloud-nuke-first-seen"
	const layout = "2006-01-02 15:04:05"

	result, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		firstSeenTime, err := getFirstSeenTag(svc, *address, key, layout)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := setFirstSeenTag(svc, *address, key, *firstSeenTime, layout); err != nil {
				return nil, err
			}
		}

		if excludeAfter.After(*firstSeenTime) {
			allocationIds = append(allocationIds, address.AllocationId)
		}
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
