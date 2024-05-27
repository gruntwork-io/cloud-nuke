package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Get all active pools
func (cs *EC2IPAMCustomAllocation) getPools() ([]*string, error) {
	result := []*string{}
	paginator := func(output *ec2.DescribeIpamPoolsOutput, lastPage bool) bool {
		for _, p := range output.IpamPools {
			result = append(result, p.IpamPoolId)
		}
		return !lastPage
	}

	params := &ec2.DescribeIpamPoolsInput{
		MaxResults: awsgo.Int64(10),
	}

	err := cs.Client.DescribeIpamPoolsPagesWithContext(cs.Context, params, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return result, nil
}

// getPoolAllocationCIDR retrieves the CIDR block associated with an IPAM pool allocation in AWS.
func (cs *EC2IPAMCustomAllocation) getPoolAllocationCIDR(allocationID *string) (*string, error) {
	// get the pool id curresponding to the allocation id
	allocationIPAMPoolID, ok := cs.PoolAndAllocationMap[*allocationID]
	if !ok {
		return nil, errors.WithStackTrace(fmt.Errorf("unable to find the pool allocation with %s", *allocationID))
	}

	output, err := cs.Client.GetIpamPoolAllocationsWithContext(cs.Context, &ec2.GetIpamPoolAllocationsInput{
		IpamPoolId:           &allocationIPAMPoolID,
		IpamPoolAllocationId: allocationID,
	})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	if !(len(output.IpamPoolAllocations) > 0) {
		return nil, errors.WithStackTrace(fmt.Errorf("unable to find the pool allocation with %s", *allocationID))
	}

	return output.IpamPoolAllocations[0].Cidr, nil
}

// Get all the allocations
func (cs *EC2IPAMCustomAllocation) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	activePools, errPool := cs.getPools()
	if errPool != nil {
		return nil, errors.WithStackTrace(errPool)
	}

	// check though the filters and see the custom allocations
	result := []*string{}
	for _, pool := range activePools {
		paginator := func(output *ec2.GetIpamPoolAllocationsOutput, lastPage bool) bool {
			for _, allocation := range output.IpamPoolAllocations {
				if *allocation.ResourceType == "custom" {
					result = append(result, allocation.IpamPoolAllocationId)
					cs.PoolAndAllocationMap[*allocation.IpamPoolAllocationId] = *pool
				}
			}
			return !lastPage
		}

		// prepare the params
		params := &ec2.GetIpamPoolAllocationsInput{
			MaxResults: awsgo.Int64(int64(cs.MaxBatchSize())),
			IpamPoolId: pool,
		}

		err := cs.Client.GetIpamPoolAllocationsPagesWithContext(cs.Context, params, paginator)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			continue
		}
	}

	// checking the nukable permissions
	cs.VerifyNukablePermissions(result, func(id *string) error {
		cidr, err := cs.getPoolAllocationCIDR(id)
		if err != nil {
			logging.Errorf("[Failed] %s", err)
			return err
		}

		allocationIPAMPoolID, ok := cs.PoolAndAllocationMap[*id]
		if !ok {
			logging.Errorf("[Failed] %s", fmt.Errorf("unable to find the pool allocation with %s", *id))
			return fmt.Errorf("unable to find the pool allocation with %s", *id)
		}

		_, err = cs.Client.ReleaseIpamPoolAllocationWithContext(cs.Context, &ec2.ReleaseIpamPoolAllocationInput{
			IpamPoolId:           &allocationIPAMPoolID,
			IpamPoolAllocationId: id,
			Cidr:                 cidr,
			DryRun:               awsgo.Bool(true),
		})
		return err
	})

	return result, nil
}

// Deletes all IPAMs
func (cs *EC2IPAMCustomAllocation) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No IPAM Custom Allocation to nuke in region %s", cs.Region)
		return nil
	}

	logging.Debugf("Deleting all IPAM Custom Allocation in region %s", cs.Region)
	var deletedAddresses []*string

	for _, id := range ids {

		if nukable, reason := cs.IsNukable(awsgo.StringValue(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(id), reason)
			continue
		}

		// get the IPamPool details
		cidr, err := cs.getPoolAllocationCIDR(id)
		if err != nil {
			logging.Errorf("[Failed] %s", err)
			continue
		}
		allocationIPAMPoolID, ok := cs.PoolAndAllocationMap[*id]
		if !ok {
			logging.Errorf("[Failed] %s", fmt.Errorf("unable to find the pool allocation with %s", *id))
			continue
		}

		// Release the allocation
		_, err = cs.Client.ReleaseIpamPoolAllocationWithContext(cs.Context, &ec2.ReleaseIpamPoolAllocationInput{
			IpamPoolId:           &allocationIPAMPoolID,
			IpamPoolAllocationId: id,
			Cidr:                 cidr,
		})

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

	logging.Debugf("[OK] %d IPAM Custom Allocation(s) deleted in %s", len(deletedAddresses), cs.Region)

	return nil
}
