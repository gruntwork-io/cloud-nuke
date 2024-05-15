package resources

import (
	"context"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func shouldIncludeIpamResourceID(ipam *ec2.IpamResourceDiscovery, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamResourceName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(ipam.Tags)
	if name, ok := tagMap["Name"]; ok {
		ipamResourceName = name
	}

	return configObj.EC2IPAMResourceDiscovery.ShouldInclude(config.ResourceValue{
		Name: &ipamResourceName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// Returns a formatted string of IPAM Resource discovery
func (discovery *EC2IPAMResourceDiscovery) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}
	var firstSeenTime *time.Time
	var err error

	paginator := func(output *ec2.DescribeIpamResourceDiscoveriesOutput, lastPage bool) bool {
		for _, d := range output.IpamResourceDiscoveries {
			firstSeenTime, err = util.GetOrCreateFirstSeen(c, discovery.Client, d.IpamResourceDiscoveryId, util.ConvertEC2TagsToMap(d.Tags))
			if err != nil {
				logging.Error("unable to retrieve firstseen tag")
				continue
			}
			if shouldIncludeIpamResourceID(d, firstSeenTime, configObj) {
				result = append(result, d.IpamResourceDiscoveryId)
			}
		}
		return !lastPage
	}

	params := &ec2.DescribeIpamResourceDiscoveriesInput{
		MaxResults: awsgo.Int64(10),
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("is-default"),
				Values: awsgo.StringSlice([]string{"false"}),
			},
		},
	}

	err = discovery.Client.DescribeIpamResourceDiscoveriesPagesWithContext(discovery.Context, params, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// checking the nukable permissions
	discovery.VerifyNukablePermissions(result, func(id *string) error {
		_, err := discovery.Client.DeleteIpamResourceDiscoveryWithContext(discovery.Context, &ec2.DeleteIpamResourceDiscoveryInput{
			IpamResourceDiscoveryId: id,
			DryRun:                  awsgo.Bool(true),
		})
		return err
	})

	return result, nil
}

// Deletes all IPAM Resource Discoveries
func (discovery *EC2IPAMResourceDiscovery) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No IPAM Resource Discovery ids to nuke in region %s", discovery.Region)
		return nil
	}

	logging.Debugf("Deleting all IPAM resource Discovery ids in region %s", discovery.Region)
	var deletedAddresses []*string

	for _, id := range ids {

		if nukable, reason := discovery.IsNukable(awsgo.StringValue(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(id), reason)
			continue
		}

		_, err := discovery.Client.DeleteIpamResourceDiscoveryWithContext(discovery.Context, &ec2.DeleteIpamResourceDiscoveryInput{
			IpamResourceDiscoveryId: id,
		})
		// Record status of this resource
		e := report.Entry{
			Identifier:   awsgo.StringValue(id),
			ResourceType: "IPAM Resource Discovery",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedAddresses = append(deletedAddresses, id)
			logging.Debugf("Deleted IPAM Resource Discovery: %s", *id)
		}
	}

	logging.Debugf("[OK] %d IPAM Resource Discovery(s) deleted in %s", len(deletedAddresses), discovery.Region)

	return nil
}
