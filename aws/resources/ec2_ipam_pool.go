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

func shouldIncludeIpamPoolID(ipam *types.IpamPool, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamPoolName string
	// get the tags as map
	tagMap := util.ConvertTypesTagsToMap(ipam.Tags)
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
	var result []*string
	var firstSeenTime *time.Time
	var err error

	params := &ec2.DescribeIpamPoolsInput{
		MaxResults: aws.Int32(10),
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"create-complete"},
			},
		},
	}

	poolsPaginator := ec2.NewDescribeIpamPoolsPaginator(ec2Pool.Client, params)
	for poolsPaginator.HasMorePages() {
		page, errPaginator := poolsPaginator.NextPage(c)
		if errPaginator != nil {
			return nil, errors.WithStackTrace(errPaginator)
		}

		for _, pool := range page.IpamPools {
			firstSeenTime, err = util.GetOrCreateFirstSeen(c, ec2Pool.Client, pool.IpamPoolId, util.ConvertTypesTagsToMap(pool.Tags))
			if err != nil {
				logging.Error("unable to retrieve first seen tag")
				continue
			}
			if shouldIncludeIpamPoolID(&pool, firstSeenTime, configObj) {
				result = append(result, pool.IpamPoolId)
			}
		}
	}

	// checking the nukable permissions
	ec2Pool.VerifyNukablePermissions(result, func(id *string) error {
		_, err := ec2Pool.Client.DeleteIpamPool(ec2Pool.Context, &ec2.DeleteIpamPoolInput{
			IpamPoolId: id,
			DryRun:     aws.Bool(true),
		})
		return err
	})

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

		if nukable, reason := pool.IsNukable(aws.ToString(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(id), reason)
			continue
		}

		_, err := pool.Client.DeleteIpamPool(pool.Context, &ec2.DeleteIpamPoolInput{
			IpamPoolId: id,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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
