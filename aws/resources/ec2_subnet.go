package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeEc2Subnet(subnet types.Subnet, firstSeenTime *time.Time, configObj config.Config) bool {
	var subnetName string
	tagMap := util.ConvertTypesTagsToMap(subnet.Tags)
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
func (ec2subnet *EC2Subnet) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}

	// Configure filters
	var filters []types.Filter
	if configObj.EC2Subnet.DefaultOnly {
		logging.Debugf("[default only] Retrieving the default subnets")
		filters = append(filters, types.Filter{
			Name:   aws.String("default-for-az"),
			Values: []string{"true"},
		})
	}

	// Create paginator
	paginator := ec2.NewDescribeSubnetsPaginator(ec2subnet.Client, &ec2.DescribeSubnetsInput{
		Filters: filters,
	})

	// Iterate through pages
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Process subnets in the current page
		for _, subnet := range page.Subnets {
			firstSeenTime, err := util.GetOrCreateFirstSeen(c, ec2subnet.Client, subnet.SubnetId, util.ConvertTypesTagsToMap(subnet.Tags))
			if err != nil {
				logging.Error("unable to retrieve first seen tag")
				continue
			}

			if shouldIncludeEc2Subnet(subnet, firstSeenTime, configObj) {
				result = append(result, subnet.SubnetId)
			}
		}
	}

	// Check if resources are nukable
	ec2subnet.VerifyNukablePermissions(result, func(id *string) error {
		params := &ec2.DeleteSubnetInput{
			SubnetId: id,
			DryRun:   aws.Bool(true), // Check permissions without making the actual request
		}
		_, err := ec2subnet.Client.DeleteSubnet(c, params)
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
		if nukable, reason := ec2subnet.IsNukable(*id); !nukable {
			// not adding the report on final result hence not adding a record entry here
			// NOTE: We can skip the error checking and return it here, since it is already being checked while
			// displaying the identifiers. Here, `err` refers to the error indicating whether the identifier is eligible for nuke or not,
			// and it is not a programming error.
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}

		err := nukeSubnet(ec2subnet.Client, id)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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

func nukeSubnet(client EC2SubnetAPI, id *string) error {
	logging.Debug(fmt.Sprintf("Deleting subnet %s",
		aws.ToString(id)))

	_, err := client.DeleteSubnet(context.Background(), &ec2.DeleteSubnetInput{
		SubnetId: id,
	})
	if err != nil {
		logging.Debug(fmt.Sprintf("Failed to delete subnet %s",
			aws.ToString(id)))
		return errors.WithStackTrace(err)
	}

	logging.Debug(fmt.Sprintf("Successfully deleted subnet %s",
		aws.ToString(id)))
	return nil
}
