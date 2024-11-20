package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Get all active pools
func (cs *EC2IPAMCustomAllocation) getPools() ([]*string, error) {
	var result []*string

	params := &ec2.DescribeIpamPoolsInput{
		MaxResults: aws.Int32(10),
	}

	poolsPaginator := ec2.NewDescribeIpamPoolsPaginator(cs.Client, params)
	for poolsPaginator.HasMorePages() {
		page, err := poolsPaginator.NextPage(cs.Context)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, p := range page.IpamPools {
			result = append(result, p.IpamPoolId)
		}
	}

	return result, nil
}

// getPoolAllocationCIDR retrieves the CIDR block associated with an IPAM pool allocation in AWS.
func (cs *EC2IPAMCustomAllocation) getPoolAllocationCIDR(allocationID *string) (*string, error) {
	// get the pool id corresponding to the allocation id
	allocationIPAMPoolID, ok := cs.PoolAndAllocationMap[*allocationID]
	if !ok {
		return nil, errors.WithStackTrace(fmt.Errorf("unable to find the pool allocation with %s", *allocationID))
	}

	output, err := cs.Client.GetIpamPoolAllocations(cs.Context, &ec2.GetIpamPoolAllocationsInput{
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
	var result []*string
	for _, pool := range activePools {
		// prepare the params
		params := &ec2.GetIpamPoolAllocationsInput{
			MaxResults: aws.Int32(int32(cs.MaxBatchSize())),
			IpamPoolId: pool,
		}

		allocationsPaginator := ec2.NewGetIpamPoolAllocationsPaginator(cs.Client, params)
		for allocationsPaginator.HasMorePages() {
			page, err := allocationsPaginator.NextPage(cs.Context)
			if err != nil {
				logging.Debugf("[Failed] %s", err)
				continue
			}

			for _, allocation := range page.IpamPoolAllocations {
				if allocation.ResourceType == types.IpamPoolAllocationResourceTypeCustom {
					result = append(result, allocation.IpamPoolAllocationId)
					cs.PoolAndAllocationMap[*allocation.IpamPoolAllocationId] = *pool
				}
			}
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

		_, err = cs.Client.ReleaseIpamPoolAllocation(cs.Context, &ec2.ReleaseIpamPoolAllocationInput{
			IpamPoolId:           &allocationIPAMPoolID,
			IpamPoolAllocationId: id,
			Cidr:                 cidr,
			DryRun:               aws.Bool(true),
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

		if nukable, reason := cs.IsNukable(aws.ToString(id)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(id), reason)
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
		_, err = cs.Client.ReleaseIpamPoolAllocation(cs.Context, &ec2.ReleaseIpamPoolAllocationInput{
			IpamPoolId:           &allocationIPAMPoolID,
			IpamPoolAllocationId: id,
			Cidr:                 cidr,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
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
