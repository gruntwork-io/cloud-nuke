package resources

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	cerrors "github.com/gruntwork-io/go-commons/errors"
)

// SecurityGroupAPI defines the interface for Security Group operations.
type SecurityGroupAPI interface {
	DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DeleteSecurityGroup(ctx context.Context, input *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error)
	RevokeSecurityGroupIngress(ctx context.Context, input *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error)
	RevokeSecurityGroupEgress(ctx context.Context, input *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error)
	DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	TerminateInstances(ctx context.Context, input *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeAddresses(ctx context.Context, input *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
}

// securityGroupResource holds extra state needed for Security Group operations.
type securityGroupResource struct {
	*resource.Resource[SecurityGroupAPI]
	nukeOnlyDefault bool
}

// NewSecurityGroup creates a new SecurityGroup resource using the generic resource pattern.
func NewSecurityGroup() AwsResource {
	r := &securityGroupResource{
		Resource: &resource.Resource[SecurityGroupAPI]{
			ResourceTypeName: "security-group",
			BatchSize:        DefaultBatchSize,
		},
	}

	r.InitClient = WrapAwsInitClient(func(res *resource.Resource[SecurityGroupAPI], cfg aws.Config) {
		res.Scope.Region = cfg.Region
		res.Client = ec2.NewFromConfig(cfg)
	})

	r.ConfigGetter = func(c config.Config) config.ResourceType {
		// Store the DefaultOnly flag for use in lister and nuker
		r.nukeOnlyDefault = c.SecurityGroup.DefaultOnly
		return c.SecurityGroup.ResourceType
	}

	r.Lister = func(ctx context.Context, client SecurityGroupAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
		return listSecurityGroups(ctx, client, cfg, r.nukeOnlyDefault)
	}

	r.Nuker = func(ctx context.Context, client SecurityGroupAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
		return nukeSecurityGroups(ctx, client, scope, resourceType, identifiers, r.nukeOnlyDefault)
	}

	r.PermissionVerifier = verifySecurityGroupNukePermission

	return &AwsResourceAdapter[SecurityGroupAPI]{Resource: r.Resource}
}

// listSecurityGroups returns security group IDs that match the filter criteria.
// Uses pagination to handle large numbers of security groups.
func listSecurityGroups(ctx context.Context, client SecurityGroupAPI, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
	var identifiers []*string

	// Build filters - for default-only mode, filter to just default security groups
	var filters []types.Filter
	if defaultOnly {
		logging.Debugf("[default only] Retrieving the default security-groups")
		filters = []types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{"default"},
			},
		}
	}

	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Security Group] Failed to list security groups: %s", err)
			return nil, cerrors.WithStackTrace(err)
		}

		for _, group := range page.SecurityGroups {
			firstSeenTime, err := util.GetOrCreateFirstSeen(ctx, client, group.GroupId, util.ConvertTypesTagsToMap(group.Tags))
			if err != nil {
				logging.Error("unable to retrieve first seen tag")
				return nil, cerrors.WithStackTrace(err)
			}

			if shouldIncludeSecurityGroup(group, firstSeenTime, cfg, defaultOnly) {
				identifiers = append(identifiers, group.GroupId)
			}
		}
	}

	return identifiers, nil
}

// shouldIncludeSecurityGroup determines whether a security group should be included for deletion.
func shouldIncludeSecurityGroup(sg types.SecurityGroup, firstSeenTime *time.Time, cfg config.ResourceType, defaultOnly bool) bool {
	groupName := sg.GroupName

	// Skip default security groups unless we're in default-only mode
	if !defaultOnly && aws.ToString(groupName) == "default" {
		logging.Debugf("[default security group] skipping default security group including")
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: groupName,
		Tags: util.ConvertTypesTagsToMap(sg.Tags),
		Time: firstSeenTime,
	})
}

// verifySecurityGroupNukePermission performs a dry-run delete to check permissions.
func verifySecurityGroupNukePermission(ctx context.Context, client SecurityGroupAPI, id *string) error {
	_, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: id,
		DryRun:  aws.Bool(true),
	})
	return util.TransformAWSError(err)
}

// nukeSecurityGroups is the custom nuker that handles both regular and default-only modes.
func nukeSecurityGroups(ctx context.Context, client SecurityGroupAPI, scope resource.Scope, resourceType string, identifiers []*string, nukeOnlyDefault bool) []resource.NukeResult {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return nil
	}
	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	results := make([]resource.NukeResult, 0, len(identifiers))
	for _, id := range identifiers {
		idStr := aws.ToString(id)
		err := nukeSecurityGroup(ctx, client, id, nukeOnlyDefault)
		results = append(results, resource.NukeResult{Identifier: idStr, Error: err})
	}

	return results
}

// nukeSecurityGroup handles the deletion of a single security group.
// This includes terminating associated instances and detaching from other security groups.
func nukeSecurityGroup(ctx context.Context, client SecurityGroupAPI, id *string, nukeOnlyDefault bool) error {
	// Step 1: Terminate instances using this security group
	if err := terminateInstancesForSecurityGroup(ctx, client, id); err != nil {
		return cerrors.WithStackTrace(err)
	}

	// Step 2: Detach from other security groups that reference this one
	if err := detachSecurityGroupReferences(ctx, client, id); err != nil {
		return cerrors.WithStackTrace(err)
	}

	// Step 3: For default security groups, just revoke rules (can't delete default SGs)
	if nukeOnlyDefault {
		return revokeAllSecurityGroupRules(ctx, client, id)
	}

	// Step 4: Delete the security group
	return deleteSecurityGroup(ctx, client, id)
}

// terminateInstancesForSecurityGroup terminates all instances using the given security group.
func terminateInstancesForSecurityGroup(ctx context.Context, client SecurityGroupAPI, id *string) error {
	resp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance.group-id"),
				Values: []string{aws.ToString(id)},
			},
		},
	})
	if err != nil {
		logging.Debugf("[terminateInstancesForSecurityGroup] Failed to describe instances: %s", err)
		return cerrors.WithStackTrace(err)
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := aws.ToString(instance.InstanceId)

			// Release elastic IPs before terminating
			if err := releaseEIPsForSecurityGroup(ctx, client, instance.InstanceId); err != nil {
				logging.Debugf("[terminateInstancesForSecurityGroup] Failed EIP release for instance %s: %s", instanceID, err)
				return cerrors.WithStackTrace(err)
			}

			// Terminate the instance
			if _, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
				InstanceIds: []string{instanceID},
			}); err != nil {
				logging.Debugf("[terminateInstancesForSecurityGroup] Failed to terminate instance %s: %s", instanceID, err)
				return cerrors.WithStackTrace(err)
			}

			// Wait for termination - need the actual EC2 client for the waiter
			ec2Client, ok := client.(*ec2.Client)
			if !ok {
				return cerrors.WithStackTrace(errors.New("cannot cast client to *ec2.Client for waiter"))
			}

			logging.Debugf("[terminateInstancesForSecurityGroup] Waiting for instance %s to terminate", instanceID)
			waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
			if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{instanceID},
			}, 5*time.Minute); err != nil {
				logging.Debugf("[terminateInstancesForSecurityGroup] Failed waiting for instance %s: %s", instanceID, err)
				return cerrors.WithStackTrace(err)
			}

			logging.Debugf("[terminateInstancesForSecurityGroup] Terminated instance %s", instanceID)
		}
	}

	return nil
}

// releaseEIPsForSecurityGroup releases all elastic IPs associated with an instance.
func releaseEIPsForSecurityGroup(ctx context.Context, client SecurityGroupAPI, instanceID *string) error {
	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []string{aws.ToString(instanceID)},
			},
		},
	})
	if err != nil {
		logging.Debugf("[releaseInstanceEIPs] Failed to describe addresses: %s", err)
		return err
	}

	for _, address := range output.Addresses {
		if _, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: address.AllocationId,
		}); err != nil {
			logging.Debugf("[releaseInstanceEIPs] Error releasing EIP %s: %v", aws.ToString(address.AllocationId), err)
			// Continue releasing other EIPs
			continue
		}
		logging.Debugf("[releaseInstanceEIPs] Released EIP %s", aws.ToString(address.AllocationId))
	}

	return nil
}

// detachSecurityGroupReferences revokes rules in other security groups that reference this one.
func detachSecurityGroupReferences(ctx context.Context, client SecurityGroupAPI, id *string) error {
	logging.Debugf("[detachSecurityGroupReferences] Detaching security group %s from dependants", aws.ToString(id))

	// Get all security groups to check for references
	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[detachSecurityGroupReferences] Failed to list security groups: %s", err)
			return cerrors.WithStackTrace(err)
		}

		for _, sg := range page.SecurityGroups {
			// Skip the security group we're deleting
			if aws.ToString(id) == aws.ToString(sg.GroupId) {
				continue
			}

			// Check and revoke ingress rules
			if hasMatch, revokePerms := findMatchingGroupRules(id, sg.IpPermissions); hasMatch && len(revokePerms) > 0 {
				logging.Debugf("[detachSecurityGroupReferences] Revoking ingress rules of %s", aws.ToString(sg.GroupId))
				if _, err := client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
					GroupId:       sg.GroupId,
					IpPermissions: revokePerms,
				}); err != nil {
					logging.Debugf("[detachSecurityGroupReferences] Failed to revoke ingress: %s", err)
					return cerrors.WithStackTrace(err)
				}
			}

			// Check and revoke egress rules
			if hasMatch, revokePerms := findMatchingGroupRules(id, sg.IpPermissionsEgress); hasMatch && len(revokePerms) > 0 {
				logging.Debugf("[detachSecurityGroupReferences] Revoking egress rules of %s", aws.ToString(sg.GroupId))
				if _, err := client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
					GroupId:       sg.GroupId,
					IpPermissions: revokePerms,
				}); err != nil {
					logging.Debugf("[detachSecurityGroupReferences] Failed to revoke egress: %s", err)
					return cerrors.WithStackTrace(err)
				}
			}
		}
	}

	return nil
}

// findMatchingGroupRules finds IP permissions that reference the given security group.
func findMatchingGroupRules(targetGroupID *string, permissions []types.IpPermission) (bool, []types.IpPermission) {
	var hasMatching bool
	var revokePermissions []types.IpPermission

	for _, perm := range permissions {
		var matchingPairs []types.UserIdGroupPair

		for _, pair := range perm.UserIdGroupPairs {
			if aws.ToString(pair.GroupId) == aws.ToString(targetGroupID) {
				matchingPairs = append(matchingPairs, pair)
				hasMatching = true
			}
		}

		if len(matchingPairs) > 0 {
			revokePerm := perm
			revokePerm.UserIdGroupPairs = matchingPairs
			revokePermissions = append(revokePermissions, revokePerm)
		}
	}

	return hasMatching, revokePermissions
}

// revokeAllSecurityGroupRules revokes all ingress and egress rules (used for default SGs).
func revokeAllSecurityGroupRules(ctx context.Context, client SecurityGroupAPI, id *string) error {
	// Revoke self-referencing ingress rule
	if err := revokeSecurityGroupIngress(ctx, client, id); err != nil {
		return cerrors.WithStackTrace(err)
	}

	// Revoke default IPv4 egress rule
	if err := revokeSecurityGroupEgress(ctx, client, id); err != nil {
		return cerrors.WithStackTrace(err)
	}

	// Revoke default IPv6 egress rule
	if err := revokeIPv6SecurityGroupEgress(ctx, client, id); err != nil {
		return cerrors.WithStackTrace(err)
	}

	return nil
}

// revokeSecurityGroupIngress revokes the default self-referencing ingress rule.
func revokeSecurityGroupIngress(ctx context.Context, client SecurityGroupAPI, id *string) error {
	logging.Debugf("[revokeSecurityGroupIngress] Revoking ingress rule: %s", aws.ToString(id))

	_, err := client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
		GroupId: id,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol:       aws.String("-1"),
				FromPort:         aws.Int32(0),
				ToPort:           aws.Int32(0),
				UserIdGroupPairs: []types.UserIdGroupPair{{GroupId: id}},
			},
		},
	})
	if err != nil {
		if errors.Is(util.TransformAWSError(err), util.ErrInvalidPermisionNotFound) {
			logging.Debugf("[revokeSecurityGroupIngress] Ingress rule not present (ok)")
			return nil
		}
		logging.Debugf("[revokeSecurityGroupIngress] Failed to revoke: %s", err)
		return cerrors.WithStackTrace(err)
	}

	logging.Debugf("[revokeSecurityGroupIngress] Successfully revoked ingress rule: %s", aws.ToString(id))
	return nil
}

// revokeSecurityGroupEgress revokes the default IPv4 egress rule.
func revokeSecurityGroupEgress(ctx context.Context, client SecurityGroupAPI, id *string) error {
	logging.Debugf("[revokeSecurityGroupEgress] Revoking egress rule: %s", aws.ToString(id))

	_, err := client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
		GroupId: id,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("-1"),
				FromPort:   aws.Int32(0),
				ToPort:     aws.Int32(0),
				IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	})
	if err != nil {
		if errors.Is(util.TransformAWSError(err), util.ErrInvalidPermisionNotFound) {
			logging.Debugf("[revokeSecurityGroupEgress] Egress rule not present (ok)")
			return nil
		}
		logging.Debugf("[revokeSecurityGroupEgress] Failed to revoke: %s", err)
		return cerrors.WithStackTrace(err)
	}

	logging.Debugf("[revokeSecurityGroupEgress] Successfully revoked egress rule: %s", aws.ToString(id))
	return nil
}

// revokeIPv6SecurityGroupEgress revokes the default IPv6 egress rule.
func revokeIPv6SecurityGroupEgress(ctx context.Context, client SecurityGroupAPI, id *string) error {
	logging.Debugf("[revokeIPv6SecurityGroupEgress] Revoking IPv6 egress rule: %s", aws.ToString(id))

	_, err := client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
		GroupId: id,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("-1"),
				FromPort:   aws.Int32(0),
				ToPort:     aws.Int32(0),
				Ipv6Ranges: []types.Ipv6Range{{CidrIpv6: aws.String("::/0")}},
			},
		},
	})
	if err != nil {
		if errors.Is(util.TransformAWSError(err), util.ErrInvalidPermisionNotFound) {
			logging.Debugf("[revokeIPv6SecurityGroupEgress] IPv6 egress rule not present (ok)")
			return nil
		}
		logging.Debugf("[revokeIPv6SecurityGroupEgress] Failed to revoke: %s", err)
		return cerrors.WithStackTrace(err)
	}

	logging.Debugf("[revokeIPv6SecurityGroupEgress] Successfully revoked IPv6 egress rule: %s", aws.ToString(id))
	return nil
}

// deleteSecurityGroup deletes a security group.
func deleteSecurityGroup(ctx context.Context, client SecurityGroupAPI, id *string) error {
	logging.Debugf("[deleteSecurityGroup] Deleting security group %s", aws.ToString(id))

	if _, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: id,
	}); err != nil {
		// Handle race condition where EKS might delete the SG before us
		if errors.Is(util.TransformAWSError(err), util.ErrInvalidGroupNotFound) {
			logging.Debugf("[deleteSecurityGroup] Security group %s already deleted (ok)", aws.ToString(id))
			return nil
		}
		logging.Debugf("[deleteSecurityGroup] Failed to delete: %s", err)
		return cerrors.WithStackTrace(err)
	}

	logging.Debugf("[deleteSecurityGroup] Successfully deleted security group %s", aws.ToString(id))
	return nil
}
