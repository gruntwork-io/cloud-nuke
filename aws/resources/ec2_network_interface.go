package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

const (
	NetworkInterfaceTypeInterface = "interface"
	// DefaultNetworkInterfaceTimeout is the default timeout for network interface operations
	DefaultNetworkInterfaceTimeout = 5 * time.Minute
)

// NetworkInterfaceAPI defines the interface for Network Interface operations.
type NetworkInterfaceAPI interface {
	DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// NewNetworkInterface creates a new Network Interface resource using the generic resource pattern.
func NewNetworkInterface() AwsResource {
	return NewAwsResource(&resource.Resource[NetworkInterfaceAPI]{
		ResourceTypeName: "network-interface",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NetworkInterfaceAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NetworkInterface
		},
		Lister:             listNetworkInterfaces,
		Nuker:              resource.SequentialDeleter(deleteNetworkInterfaceWithDetach),
		PermissionVerifier: verifyNetworkInterfacePermission,
	})
}

// listNetworkInterfaces retrieves all Network Interfaces that match the config filters.
func listNetworkInterfaces(ctx context.Context, client NetworkInterfaceAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var interfaceIds []*string

	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Network Interface] Failed to list network interfaces: %s", err)
			return nil, err
		}

		for _, networkInterface := range page.NetworkInterfaces {
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, networkInterface.NetworkInterfaceId, util.ConvertTypesTagsToMap(networkInterface.TagSet))
			if err != nil {
				logging.Errorf("[Network Interface] Unable to retrieve first seen tag for %s: %s", aws.ToString(networkInterface.NetworkInterfaceId), err)
				continue
			}

			// NOTE: Not all network interface types can be detached programmatically, and some may take longer to nuke.
			// Interfaces attached to Lambda or other AWS services may have specific detachment mechanisms managed by
			// those services. Attempting to detach these via the API can cause errors. Skipping non-interface types
			// ensures they are cleaned up automatically upon service deletion.
			if networkInterface.InterfaceType != NetworkInterfaceTypeInterface {
				logging.Debugf("[Skip] Can't detach network interface of type '%v' via API. "+
					"Detachment for this type is managed by the dependent service and will occur automatically upon "+
					"resource deletion.", networkInterface.InterfaceType)
				continue
			}

			if shouldIncludeNetworkInterface(networkInterface, firstSeenTime, cfg) {
				interfaceIds = append(interfaceIds, networkInterface.NetworkInterfaceId)
			}
		}
	}

	return interfaceIds, nil
}

// shouldIncludeNetworkInterface checks if a network interface should be included based on config filters.
func shouldIncludeNetworkInterface(networkInterface types.NetworkInterface, firstSeenTime *time.Time, cfg config.ResourceType) bool {
	var interfaceName string
	tagMap := util.ConvertTypesTagsToMap(networkInterface.TagSet)
	if name, ok := tagMap["Name"]; ok {
		interfaceName = name
	}
	return cfg.ShouldInclude(config.ResourceValue{
		Name: &interfaceName,
		Tags: tagMap,
		Time: firstSeenTime,
	})
}

// verifyNetworkInterfacePermission performs a dry-run delete to check permissions.
func verifyNetworkInterfacePermission(ctx context.Context, client NetworkInterfaceAPI, id *string) error {
	_, err := client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: id,
		DryRun:             aws.Bool(true),
	})
	return err
}

// deleteNetworkInterfaceWithDetach detaches and deletes a single Network Interface.
// This is the function used by the batch deleter.
func deleteNetworkInterfaceWithDetach(ctx context.Context, client NetworkInterfaceAPI, id *string) error {
	// First detach the network interface if attached
	if err := detachNetworkInterface(ctx, client, id); err != nil {
		return err
	}

	// Then delete the network interface
	return deleteNetworkInterfaceByID(ctx, client, id)
}

// detachNetworkInterface detaches a network interface by terminating any attached instances.
func detachNetworkInterface(ctx context.Context, client NetworkInterfaceAPI, id *string) error {
	if id == nil || aws.ToString(id) == "" {
		logging.Debugf("[detachNetworkInterface] Network interface ID is nil or empty, skipping detachment process")
		return nil
	}

	logging.Debugf("[detachNetworkInterface] Detaching network interface %s from instances", aws.ToString(id))

	output, err := client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{aws.ToString(id)},
	})
	if err != nil {
		logging.Debugf("[detachNetworkInterface] Failed to describe network interface %s: %v", aws.ToString(id), err)
		return err
	}

	for _, networkInterface := range output.NetworkInterfaces {
		if networkInterface.Attachment == nil || networkInterface.Attachment.InstanceId == nil {
			logging.Debugf("[detachNetworkInterface] No attachment found for network interface %s, skipping", aws.ToString(id))
			continue
		}

		instanceID := networkInterface.Attachment.InstanceId
		logging.Debugf("[detachNetworkInterface] Found attached instance %s for network interface %s", aws.ToString(instanceID), aws.ToString(id))

		if err := nukeAttachedInstance(ctx, client, instanceID); err != nil {
			logging.Debugf("[detachNetworkInterface] Failed to nuke instance %s attached to network interface %s: %v", aws.ToString(instanceID), aws.ToString(id), err)
			return err
		}

		logging.Debugf("[detachNetworkInterface] Successfully nuked instance %s and detached network interface %s", aws.ToString(instanceID), aws.ToString(id))
	}

	return nil
}

// nukeAttachedInstance releases EIPs and terminates an instance attached to a network interface.
func nukeAttachedInstance(ctx context.Context, client NetworkInterfaceAPI, instanceID *string) error {
	if instanceID == nil || aws.ToString(instanceID) == "" {
		return nil
	}

	// Release the elastic IPs attached to the instance before nuking
	if err := releaseNetworkInterfaceEIPs(ctx, client, instanceID); err != nil {
		logging.Debugf("[nukeAttachedInstance] Failed to release Elastic IPs for instance %s: %v", aws.ToString(instanceID), err)
		return err
	}

	// Terminate the instance
	_, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{aws.ToString(instanceID)},
	})
	if err != nil {
		logging.Debugf("[nukeAttachedInstance] Failed to terminate instance %s: %v", aws.ToString(instanceID), err)
		return err
	}

	logging.Debugf("[nukeAttachedInstance] Waiting for instance %s to terminate", aws.ToString(instanceID))

	// Wait for instance to be terminated
	waiter := ec2.NewInstanceTerminatedWaiter(client)
	if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{aws.ToString(instanceID)},
	}, DefaultNetworkInterfaceTimeout); err != nil {
		logging.Debugf("[nukeAttachedInstance] Instance termination waiting failed for instance %s: %v", aws.ToString(instanceID), err)
		return err
	}

	logging.Debugf("[nukeAttachedInstance] Successfully nuked instance %s", aws.ToString(instanceID))
	return nil
}

// releaseNetworkInterfaceEIPs releases all Elastic IPs associated with an instance.
// This is specific to network interface cleanup and uses the NetworkInterfaceAPI.
func releaseNetworkInterfaceEIPs(ctx context.Context, client NetworkInterfaceAPI, instanceID *string) error {
	if instanceID == nil || aws.ToString(instanceID) == "" {
		return nil
	}

	logging.Debugf("[releaseNetworkInterfaceEIPs] Releasing Elastic IP address(es) associated with instance %s", aws.ToString(instanceID))

	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []string{aws.ToString(instanceID)},
			},
		},
	})
	if err != nil {
		logging.Debugf("[releaseNetworkInterfaceEIPs] Failed to describe addresses for instance %s: %v", aws.ToString(instanceID), err)
		return err
	}

	for _, address := range output.Addresses {
		if address.AllocationId == nil {
			continue
		}

		if _, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: address.AllocationId,
		}); err != nil {
			logging.Debugf("[releaseNetworkInterfaceEIPs] Failed to release Elastic IP address %s for instance %s: %v",
				aws.ToString(address.AllocationId), aws.ToString(instanceID), err)
			continue
		}

		logging.Debugf("[releaseNetworkInterfaceEIPs] Successfully released Elastic IP address %s from instance %s",
			aws.ToString(address.AllocationId), aws.ToString(instanceID))
	}

	return nil
}

// deleteNetworkInterfaceByID deletes a single network interface.
func deleteNetworkInterfaceByID(ctx context.Context, client NetworkInterfaceAPI, id *string) error {
	logging.Debugf("Deleting network interface %s", aws.ToString(id))

	// If the network interface was attached to an instance, then when we remove the instance above, the network interface will also be removed.
	// However, when we attempt to nuke the interface here, we may encounter an error such as InvalidNetworkInterfaceID.NotFound.
	// In other situations, such as when the network interface hasn't been attached to any instances, we won't encounter this error.
	//
	// Note: We are handling the situation here by checking the error response from AWS.
	_, err := client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: id,
	})

	// Check if the error is not the interface ID not found error
	if err != nil && util.TransformAWSError(err) != util.ErrInterfaceIDNotFound {
		logging.Debugf("[Failed] Error deleting network interface %s: %s", aws.ToString(id), err)
		return err
	}

	logging.Debugf("[Ok] network interface deleted successfully %s", aws.ToString(id))
	return nil
}
