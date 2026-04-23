package resources

import (
	"context"
	"fmt"

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

type allocationInfo struct {
	PoolID string
	Cidr   string
}

// NewEC2IPAMCustomAllocation creates a new EC2 IPAM Custom Allocation resource using the generic resource pattern.
// Pool+CIDR metadata discovered during listing is required again at deletion time (ReleaseIpamPoolAllocation needs
// both), so it is stashed in a per-instance map captured by closure. Per-instance scoping matters because
// GetAndInitRegisteredResources creates a fresh instance for each region — sharing one map across regions would
// let later regions' Init calls overwrite earlier regions' metadata and break deletion.
func NewEC2IPAMCustomAllocation() AwsResource {
	poolAndAllocationMap := make(map[string]allocationInfo)

	return NewAwsResource(&resource.Resource[EC2IPAMCustomAllocationAPI]{
		ResourceTypeName: "ipam-custom-allocation",
		BatchSize:        1000,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EC2IPAMCustomAllocationAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EC2IPAMCustomAllocation
		},
		Lister: func(ctx context.Context, client EC2IPAMCustomAllocationAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listEC2IPAMCustomAllocations(ctx, client, poolAndAllocationMap)
		},
		Nuker: resource.SimpleBatchDeleter(func(ctx context.Context, client EC2IPAMCustomAllocationAPI, id *string) error {
			return deleteEC2IPAMCustomAllocation(ctx, client, id, poolAndAllocationMap)
		}),
		PermissionVerifier: func(ctx context.Context, client EC2IPAMCustomAllocationAPI, id *string) error {
			return verifyEC2IPAMCustomAllocationPermission(ctx, client, id, poolAndAllocationMap)
		},
	})
}

// listEC2IPAMCustomAllocations retrieves all custom allocations across all IPAM pools.
func listEC2IPAMCustomAllocations(ctx context.Context, client EC2IPAMCustomAllocationAPI, poolAndAllocationMap map[string]allocationInfo) ([]*string, error) {
	pools, err := getPools(ctx, client)
	if err != nil {
		return nil, err
	}

	var result []*string

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
					poolAndAllocationMap[aws.ToString(allocation.IpamPoolAllocationId)] = allocationInfo{
						PoolID: aws.ToString(poolID),
						Cidr:   aws.ToString(allocation.Cidr),
					}
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
func verifyEC2IPAMCustomAllocationPermission(ctx context.Context, client EC2IPAMCustomAllocationAPI, id *string, poolAndAllocationMap map[string]allocationInfo) error {
	info, ok := poolAndAllocationMap[aws.ToString(id)]
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
func deleteEC2IPAMCustomAllocation(ctx context.Context, client EC2IPAMCustomAllocationAPI, id *string, poolAndAllocationMap map[string]allocationInfo) error {
	info, ok := poolAndAllocationMap[aws.ToString(id)]
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
