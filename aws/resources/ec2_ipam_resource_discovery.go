package resources

import (
	"context"
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

func shouldIncludeIpamResourceID(ipam *types.IpamResourceDiscovery, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamResourceName string
	// get the tags as map
	tagMap := util.ConvertTypesTagsToMap(ipam.Tags)
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
	var result []*string
	var firstSeenTime *time.Time
	var err error

	params := &ec2.DescribeIpamResourceDiscoveriesInput{
		MaxResults: aws.Int32(10),
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"false"},
			},
		},
	}

	discoveriesPaginator := ec2.NewDescribeIpamResourceDiscoveriesPaginator(discovery.Client, params)
	for discoveriesPaginator.HasMorePages() {
		page, errPaginator := discoveriesPaginator.NextPage(c)
		if errPaginator != nil {
			return nil, errors.WithStackTrace(errPaginator)
		}

		for _, d := range page.IpamResourceDiscoveries {
			firstSeenTime, err = util.GetOrCreateFirstSeen(c, discovery.Client, d.IpamResourceDiscoveryId, util.ConvertTypesTagsToMap(d.Tags))
			if err != nil {
				logging.Error("unable to retrieve firstseen tag")
				continue
			}
			if shouldIncludeIpamResourceID(&d, firstSeenTime, configObj) {
				result = append(result, d.IpamResourceDiscoveryId)
			}
		}

	}

	// checking the nukable permissions
	discovery.VerifyNukablePermissions(result, func(id *string) error {
		_, err := discovery.Client.DeleteIpamResourceDiscovery(discovery.Context, &ec2.DeleteIpamResourceDiscoveryInput{
			IpamResourceDiscoveryId: id,
			DryRun:                  aws.Bool(true),
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

		if nukable, reason := discovery.IsNukable(aws.ToString(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(id), reason)
			continue
		}

		_, err := discovery.Client.DeleteIpamResourceDiscovery(discovery.Context, &ec2.DeleteIpamResourceDiscoveryInput{
			IpamResourceDiscoveryId: id,
		})
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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
