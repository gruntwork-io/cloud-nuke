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

func (discovery *EC2IPAMResourceDiscovery) setFirstSeenTag(ipam ec2.IpamResourceDiscovery, value time.Time) error {
	_, err := discovery.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{ipam.IpamResourceDiscoveryId},
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

func (discovery *EC2IPAMResourceDiscovery) getFirstSeenTag(ipam ec2.IpamResourceDiscovery) (*time.Time, error) {
	tags := ipam.Tags
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
	paginator := func(output *ec2.DescribeIpamResourceDiscoveriesOutput, lastPage bool) bool {
		for _, d := range output.IpamResourceDiscoveries {
			// check first seen tag
			firstSeenTime, err := discovery.getFirstSeenTag(*d)
			if err != nil {
				logging.Errorf(
					"Unable to retrieve tags for IPAM: %s, with error: %s", *d.IpamResourceDiscoveryId, err)
				continue
			}

			// if the first seen tag is not there, then create one
			if firstSeenTime == nil {
				now := time.Now().UTC()
				firstSeenTime = &now
				if err := discovery.setFirstSeenTag(*d, time.Now().UTC()); err != nil {
					logging.Errorf(
						"Unable to apply first seen tag IPAM: %s, with error: %s", *d.IpamResourceDiscoveryId, err)
					continue
				}
			}
			// Check for include this ipam resource ID
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

	err := discovery.Client.DescribeIpamResourceDiscoveriesPages(params, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

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
		params := &ec2.DeleteIpamResourceDiscoveryInput{
			IpamResourceDiscoveryId: id,
		}

		_, err := discovery.Client.DeleteIpamResourceDiscovery(params)
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
