package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
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
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			return &firstSeenTime, nil
		}
	}

	return nil, nil
}

// Returns a formatted string of EIP allocation ids
func getAllEIPAddresses(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)
	const layout = "2006-01-02 15:04:05"

	result, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var allocationIds []*string
	for _, address := range result.Addresses {
		firstSeenTime, err := getFirstSeenTag(svc, *address, firstSeenTagKey, layout)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if firstSeenTime == nil {
			now := time.Now().UTC()
			firstSeenTime = &now
			if err := setFirstSeenTag(svc, *address, firstSeenTagKey, *firstSeenTime, layout); err != nil {
				return nil, err
			}
		}
		if shouldIncludeAllocationId(address, excludeAfter, *firstSeenTime, configObj) {
			allocationIds = append(allocationIds, address.AllocationId)
		}
	}

	return allocationIds, nil
}

// hasEIPExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasEIPExcludeTag(address *ec2.Address) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	for _, tag := range address.Tags {
		if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
			return true
		}
	}
	return false
}

func shouldIncludeAllocationId(address *ec2.Address, excludeAfter time.Time, firstSeenTime time.Time, configObj config.Config) bool {
	if address == nil {
		return false
	}

	if excludeAfter.Before(firstSeenTime) {
		return false
	}

	if hasEIPExcludeTag(address) {
		return false
	}

	// If Name is unset, GetEC2ResourceNameTagValue returns error and zero value string
	// Ignore this error and pass empty string to config.ShouldInclude
	allocationName, _ := GetEC2ResourceNameTagValue(address.Tags)

	return config.ShouldInclude(
		allocationName,
		configObj.ElasticIP.IncludeRule.NamesRegExp,
		configObj.ElasticIP.ExcludeRule.NamesRegExp,
	)
}

// Deletes all EIP allocation ids
func nukeAllEIPAddresses(session *session.Session, allocationIds []*string) error {
	svc := ec2.New(session)

	if len(allocationIds) == 0 {
		logging.Logger.Debugf("No Elastic IPs to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Elastic IPs in region %s", *session.Config.Region)
	var deletedAllocationIDs []*string

	for _, allocationID := range allocationIds {
		params := &ec2.ReleaseAddressInput{
			AllocationId: allocationID,
		}

		_, err := svc.ReleaseAddress(params)

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
				logging.Logger.Debugf("EIP %s can't be deleted, it is still attached to an active resource", *allocationID)
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EIP",
				}, map[string]interface{}{
					"region": *session.Config.Region,
					"reason": "Still Attached to an Active Resource",
				})
			} else {
				logging.Logger.Debugf("[Failed] %s", err)
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EIP",
				}, map[string]interface{}{
					"region": *session.Config.Region,
				})
			}
		} else {
			deletedAllocationIDs = append(deletedAllocationIDs, allocationID)
			logging.Logger.Debugf("Deleted Elastic IP: %s", *allocationID)
		}
	}

	logging.Logger.Debugf("[OK] %d Elastic IP(s) deleted in %s", len(deletedAllocationIDs), *session.Config.Region)
	return nil
}
