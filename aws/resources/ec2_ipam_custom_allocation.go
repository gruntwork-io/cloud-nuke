package resources

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EC2IPAMCustomAllocationAPI defines the interface for EC2 IPAM Custom Allocation operations.
type EC2IPAMCustomAllocationAPI interface {
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error)
	ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error)
}

// ipamCustomAllocationState holds runtime state needed for custom allocation operations.
// This is stored alongside the Resource to track pool-allocation mappings during listing and deletion.
type ipamCustomAllocationState struct {
	mu                   sync.RWMutex
	poolAndAllocationMap map[string]allocationInfo
}

type allocationInfo struct {
	PoolID string
	Cidr   string
}

var customAllocationState = &ipamCustomAllocationState{
	poolAndAllocationMap: make(map[string]allocationInfo),
}

// NewEC2IPAMCustomAllocation creates a new EC2 IPAM Custom Allocation resource using the generic resource pattern.
func NewEC2IPAMCustomAllocation() AwsResource {
	return NewAwsResource(&resource.Resource[EC2IPAMCustomAllocationAPI]{
		ResourceTypeName: "ipam-custom-allocation",
		BatchSize:        1000,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMCustomAllocationAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
			// Reset state on init
			customAllocationState.mu.Lock()
			customAllocationState.poolAndAllocationMap = make(map[string]allocationInfo)
			customAllocationState.mu.Unlock()
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAMCustomAllocation
		},
		Lister:             listEC2IPAMCustomAllocations,
		Nuker:              resource.SimpleBatchDeleter(deleteEC2IPAMCustomAllocation),
		PermissionVerifier: verifyEC2IPAMCustomAllocationPermission,
	})
}

// listEC2IPAMCustomAllocations retrieves all custom allocations across all IPAM pools.
func listEC2IPAMCustomAllocations(ctx context.Context, client EC2IPAMCustomAllocationAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// First, get all pools
	pools, err := getPools(ctx, client)
	if err != nil {
		return nil, err
	}

	var result []*string

	// For each pool, get all custom allocations
	for _, poolID := range pools {
		paginator := ec2.NewGetIpamPoolAllocationsPaginator(client, &ec2.GetIpamPoolAllocationsInput{
			MaxResults: aws.Int32(1000),
			IpamPoolId: poolID,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				logging.Debugf("Failed to get allocations for pool %s: %v", aws.ToString(poolID), err)
				break
			}

			for _, allocation := range page.IpamPoolAllocations {
				if allocation.ResourceType == types.IpamPoolAllocationResourceTypeCustom {
					result = append(result, allocation.IpamPoolAllocationId)
					// Store pool and CIDR info for later use during deletion
					customAllocationState.mu.Lock()
					customAllocationState.poolAndAllocationMap[aws.ToString(allocation.IpamPoolAllocationId)] = allocationInfo{
						PoolID: aws.ToString(poolID),
						Cidr:   aws.ToString(allocation.Cidr),
					}
					customAllocationState.mu.Unlock()
				}
			}
		}
	}

	return result, nil
}

// getPools retrieves all IPAM pools.
func getPools(ctx context.Context, client EC2IPAMCustomAllocationAPI) ([]*string, error) {
	var result []*string

	paginator := ec2.NewDescribeIpamPoolsPaginator(client, &ec2.DescribeIpamPoolsInput{
		MaxResults: aws.Int32(10),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, pool := range page.IpamPools {
			result = append(result, pool.IpamPoolId)
		}
	}

	return result, nil
}

// verifyEC2IPAMCustomAllocationPermission performs a dry-run release to check permissions.
func verifyEC2IPAMCustomAllocationPermission(ctx context.Context, client EC2IPAMCustomAllocationAPI, id *string) error {
	customAllocationState.mu.RLock()
	info, ok := customAllocationState.poolAndAllocationMap[aws.ToString(id)]
	customAllocationState.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unable to find pool allocation info for %s", aws.ToString(id))
	}

	_, err := client.ReleaseIpamPoolAllocation(ctx, &ec2.ReleaseIpamPoolAllocationInput{
		IpamPoolId:           aws.String(info.PoolID),
		IpamPoolAllocationId: id,
		Cidr:                 aws.String(info.Cidr),
		DryRun:               aws.Bool(true),
	})
	return err
}

// deleteEC2IPAMCustomAllocation releases a single custom allocation.
func deleteEC2IPAMCustomAllocation(ctx context.Context, client EC2IPAMCustomAllocationAPI, id *string) error {
	customAllocationState.mu.RLock()
	info, ok := customAllocationState.poolAndAllocationMap[aws.ToString(id)]
	customAllocationState.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unable to find pool allocation info for %s", aws.ToString(id))
	}

	_, err := client.ReleaseIpamPoolAllocation(ctx, &ec2.ReleaseIpamPoolAllocationInput{
		IpamPoolId:           aws.String(info.PoolID),
		IpamPoolAllocationId: id,
		Cidr:                 aws.String(info.Cidr),
	})
	return err
}
