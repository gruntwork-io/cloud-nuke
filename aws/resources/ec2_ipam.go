package resources

import (
	"context"
	"fmt"
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

func shouldIncludeIpamID(ipam *ec2.Ipam, firstSeenTime *time.Time, configObj config.Config) bool {
	var ipamName string
	// get the tags as map
	tagMap := util.ConvertEC2TagsToMap(ipam.Tags)
	if name, ok := tagMap["Name"]; ok {
		ipamName = name
	}

	return configObj.EC2IPAM.ShouldInclude(config.ResourceValue{
		Name: &ipamName,
		Time: firstSeenTime,
		Tags: tagMap,
	})
}

// Returns a formatted string of IPAM URLs
func (ec2Ipam *EC2IPAMs) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result := []*string{}
	var firstSeenTime *time.Time
	var err error

	paginator := func(output *ec2.DescribeIpamsOutput, lastPage bool) bool {
		for _, ipam := range output.Ipams {
			firstSeenTime, err = util.GetOrCreateFirstSeen(c, ec2Ipam.Client, ipam.IpamId, util.ConvertEC2TagsToMap(ipam.Tags))
			if err != nil {
				logging.Error("Unable to retrieve tags")
				continue
			}
			// Check for include this ipam
			if shouldIncludeIpamID(ipam, firstSeenTime, configObj) {
				result = append(result, ipam.IpamId)
			}
		}
		return !lastPage
	}

	params := &ec2.DescribeIpamsInput{
		MaxResults: awsgo.Int64(10),
	}

	err = ec2Ipam.Client.DescribeIpamsPages(params, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// checking the nukable permissions
	ec2Ipam.VerifyNukablePermissions(result, func(id *string) error {
		_, err := ec2Ipam.Client.DeleteIpam(&ec2.DeleteIpamInput{
			IpamId:  id,
			Cascade: aws.Bool(true),
			DryRun:  awsgo.Bool(true),
		})
		return err
	})

	return result, nil
}

// deProvisionPoolCIDRs : Detach the CIDR provisiond on the pool
func (ec2Ipam *EC2IPAMs) deProvisionPoolCIDRs(poolID *string) error {
	output, err := ec2Ipam.Client.GetIpamPoolCidrs(&ec2.GetIpamPoolCidrsInput{
		IpamPoolId: poolID,
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("state"),
				Values: awsgo.StringSlice([]string{"provisioned"}),
			},
		},
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, poolCidr := range output.IpamPoolCidrs {
		_, err := ec2Ipam.Client.DeprovisionIpamPoolCidr(&ec2.DeprovisionIpamPoolCidrInput{
			IpamPoolId: poolID,
			Cidr:       poolCidr.Cidr,
		})

		if err != nil {
			return errors.WithStackTrace(err)
		}
		logging.Debugf("De-Provisioned CIDR(s) from IPAM Pool %s", aws.StringValue(poolID))
	}

	return nil
}

// releaseCustomAllocations : Release the custom allocated CIDR(s) from the pool
func (ec2Ipam *EC2IPAMs) releaseCustomAllocations(poolID *string) error {
	output, err := ec2Ipam.Client.GetIpamPoolAllocations(&ec2.GetIpamPoolAllocationsInput{
		IpamPoolId: poolID,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, poolAllocation := range output.IpamPoolAllocations {
		// we only can release the custom allocations
		if *poolAllocation.ResourceType != "custom" {
			continue
		}
		_, err := ec2Ipam.Client.ReleaseIpamPoolAllocation(&ec2.ReleaseIpamPoolAllocationInput{
			IpamPoolId:           poolID,
			IpamPoolAllocationId: poolAllocation.IpamPoolAllocationId,
			Cidr:                 poolAllocation.Cidr,
		})

		if err != nil {
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Release custom allocated CIDR(s) from IPAM Pool %s", aws.StringValue(poolID))
	}

	return nil
}

// nukePublicIPAMPools : Nuke the pools on an IPAM
// Before deleting the IPAM, it is necessary to manually remove any pools within our public scope,
// as the deleteIPAM operation will not handle their deletion with cascade option.
//
// We cannot delete an IPAM pool if there are allocations in it or CIDRs provisioned to it. We must first release the allocations and Deprovision CIDRs
// from a pool before we can delete the pool
func (ec2Ipam *EC2IPAMs) nukePublicIPAMPools(ipamID *string) error {
	ipam, err := ec2Ipam.Client.DescribeIpams(&ec2.DescribeIpamsInput{
		IpamIds: aws.StringSlice([]string{*ipamID}),
	})
	if err != nil {
		logging.Errorf(fmt.Sprintf("Error describing IPAM %s: %s", *ipamID, err.Error()))
		return errors.WithStackTrace(err)
	}

	// Describe the scope to read the scope arn
	scope, err := ec2Ipam.Client.DescribeIpamScopes(&ec2.DescribeIpamScopesInput{
		IpamScopeIds: aws.StringSlice([]string{
			*ipam.Ipams[0].PublicDefaultScopeId,
		}),
	})

	if err != nil {
		logging.Errorf(fmt.Sprintf("Error describing IPAM Public scope %s: %s", *ipamID, err.Error()))
		return errors.WithStackTrace(err)
	}

	// get the pools which is assigned on the public scope of the IPAM
	output, err := ec2Ipam.Client.DescribeIpamPools(&ec2.DescribeIpamPoolsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("ipam-scope-arn"),
				Values: aws.StringSlice([]string{
					*scope.IpamScopes[0].IpamScopeArn,
				}),
			},
		},
	})
	if err != nil {
		logging.Errorf(fmt.Sprintf("Error describing IPAM Pools on public scope %s: %s", *ipamID, err.Error()))
		return errors.WithStackTrace(err)
	}

	for _, pool := range output.IpamPools {
		// Remove associated CIDRs before deleting IPAM pools to complete de-provisioning.
		err := ec2Ipam.deProvisionPoolCIDRs(pool.IpamPoolId)
		if err != nil {
			logging.Errorf(fmt.Sprintf("Error de-provisioning Pools CIDR  on Pool %s : %s", *pool.IpamPoolId, err.Error()))
			return errors.WithStackTrace(err)
		}

		// Release custom allocation from the pool
		err = ec2Ipam.releaseCustomAllocations(pool.IpamPoolId)
		if err != nil {
			logging.Errorf(fmt.Sprintf("Error Release custom allocations of Pool %s : %s", *pool.IpamPoolId, err.Error()))
			return errors.WithStackTrace(err)
		}

		// delete ipam pool
		_, err = ec2Ipam.Client.DeleteIpamPool(&ec2.DeleteIpamPoolInput{
			IpamPoolId: pool.IpamPoolId,
		})
		if err != nil {
			logging.Errorf("[Failed] Delete IPAM Pool %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Deleted IPAM Pool %s from IPAM %s", aws.StringValue(pool.IpamPoolId), aws.StringValue(ipamID))
	}

	return nil
}

// deleteIPAM : Delete the IPAM
func (ec2Ipam *EC2IPAMs) deleteIPAM(id *string) error {
	params := &ec2.DeleteIpamInput{
		IpamId: id,
		// NOTE : Enables you to quickly delete an IPAM, private scopes, pools in private scopes, and any allocations in the pools in private scopes.
		// You cannot delete the IPAM with this option if there is a pool in your public scope.
		// IPAM does the following when this is enabled
		//
		// * Deallocates any CIDRs allocated to VPC resources (such as VPCs) in pools
		// * Deprovisions all IPv4 CIDRs provisioned to IPAM pools in private scopes.
		// * Deletes all IPAM pools in private scopes.
		// * Deletes all non-default private scopes in the IPAM.
		// * Deletes the default public and private scopes and the IPAM.
		Cascade: aws.Bool(true),
	}

	_, err := ec2Ipam.Client.DeleteIpam(params)

	return err
}

func (ec2Ipam *EC2IPAMs) nukeIPAM(id *string) error {
	// Functions used to really nuke an IPAM as an IPAM can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual IPAM deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(*string) error{
		ec2Ipam.nukePublicIPAMPools,
		ec2Ipam.deleteIPAM,
	}

	for _, fn := range functions {
		if err := fn(id); err != nil {
			return err
		}
	}

	return nil
}

// Deletes all IPAMs
func (ec2Ipam *EC2IPAMs) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No IPAM ids to nuke in region %s", ec2Ipam.Region)
		return nil
	}

	logging.Debugf("Deleting all IPAM ids in region %s", ec2Ipam.Region)
	var deletedAddresses []*string

	for _, id := range ids {

		if nukable, reason := ec2Ipam.IsNukable(awsgo.StringValue(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(id), reason)
			continue
		}

		err := ec2Ipam.nukeIPAM(id)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "IPAM",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedAddresses = append(deletedAddresses, id)
			logging.Debugf("Deleted IPAM: %s", *id)
		}
	}

	logging.Debugf("[OK] %d IPAM address(s) deleted in %s", len(deletedAddresses), ec2Ipam.Region)

	return nil
}
