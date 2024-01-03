package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (discovery *EC2IPAMPool) setFirstSeenTag(ipam ec2.IpamPool, value time.Time) error {
	_, err := discovery.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{ipam.IpamPoolId},
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

func (discovery *EC2IPAMPool) getFirstSeenTag(ipam ec2.IpamPool) (*time.Time, error) {
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

func shouldIncludeIpamPoolID(ipam *ec2.IpamPool, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamPoolName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(ipam.Tags)
	if name, ok := tagMap["Name"]; ok {
		ipamPoolName = name
	}

	return configObj.EC2IPAMPool.ShouldInclude(config.ResourceValue{
		Name: &ipamPoolName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// Returns a formatted string of IPAM URLs
func (ec2Pool *EC2IPAMPool) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}
	paginator := func(output *ec2.DescribeIpamPoolsOutput, lastPage bool) bool {
		for _, pool := range output.IpamPools {
			// check first seen tag
			firstSeenTime, err := ec2Pool.getFirstSeenTag(*pool)
			if err != nil {
				logging.Errorf(
					"Unable to retrieve tags for IPAM Pool: %s, with error: %s", *pool.IpamPoolId, err)
				continue
			}

			// if the first seen tag is not there, then create one
			if firstSeenTime == nil {
				now := time.Now().UTC()
				firstSeenTime = &now
				if err := ec2Pool.setFirstSeenTag(*pool, time.Now().UTC()); err != nil {
					logging.Errorf(
						"Unable to apply first seen tag IPAM Pool: %s, with error: %s", *pool.IpamPoolId, err)
					continue
				}
			}
			// Check for include this ipam
			if shouldIncludeIpamPoolID(pool, firstSeenTime, configObj) {
				result = append(result, pool.IpamPoolId)
			}
		}
		return !lastPage
	}

	params := &ec2.DescribeIpamPoolsInput{
		MaxResults: awsgo.Int64(10),
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("state"),
				Values: awsgo.StringSlice([]string{"create-complete"}),
			},
		},
	}

	err := ec2Pool.Client.DescribeIpamPoolsPages(params, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return result, nil
}

// Deletes all IPAMs
func (pool *EC2IPAMPool) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No IPAM ids to nuke in region %s", pool.Region)
		return nil
	}

	logging.Debugf("Deleting all IPAM ids in region %s", pool.Region)
	var deletedAddresses []*string

	for _, id := range ids {
		params := &ec2.DeleteIpamPoolInput{
			IpamPoolId: id,
		}

		_, err := pool.Client.DeleteIpamPool(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "IPAM Pool",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedAddresses = append(deletedAddresses, id)
			logging.Debugf("Deleted IPAM Pool: %s", *id)
		}
	}

	logging.Debugf("[OK] %d IPAM Pool(s) deleted in %s", len(deletedAddresses), pool.Region)

	return nil
}
