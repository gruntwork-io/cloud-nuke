package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// DBGlobalClusterMembershipsAPI defines the interface for RDS Global Cluster Membership operations.
type DBGlobalClusterMembershipsAPI interface {
	DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error)
	RemoveFromGlobalCluster(ctx context.Context, params *rds.RemoveFromGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.RemoveFromGlobalClusterOutput, error)
}

// NewDBGlobalClusterMemberships creates a new RDS Global Cluster Memberships resource.
func NewDBGlobalClusterMemberships() AwsResource {
	return NewAwsResource(&resource.Resource[DBGlobalClusterMembershipsAPI]{
		ResourceTypeName: "rds-global-cluster-membership",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DBGlobalClusterMembershipsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBGlobalClusterMemberships
		},
		Lister: listDBGlobalClusterMemberships,
		Nuker:  nukeDBGlobalClusterMemberships,
	})
}

// listDBGlobalClusterMemberships retrieves all RDS Global Clusters that have members.
func listDBGlobalClusterMemberships(ctx context.Context, client DBGlobalClusterMembershipsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string
	paginator := rds.NewDescribeGlobalClustersPaginator(client, &rds.DescribeGlobalClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range page.GlobalClusters {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: cluster.GlobalClusterIdentifier,
			}) {
				names = append(names, cluster.GlobalClusterIdentifier)
			}
		}
	}

	return names, nil
}

// nukeDBGlobalClusterMemberships removes cluster memberships from global clusters.
// This is a custom nuker because it needs region context to filter which members to remove.
func nukeDBGlobalClusterMemberships(ctx context.Context, client DBGlobalClusterMembershipsAPI, scope resource.Scope, resourceType string, identifiers []*string) []resource.NukeResult {
	if len(identifiers) == 0 {
		logging.Debugf("No RDS Global Cluster Memberships to nuke")
		return nil
	}

	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	results := make([]resource.NukeResult, 0, len(identifiers))
	for _, name := range identifiers {
		idStr := aws.ToString(name)
		err := removeGlobalClusterMemberships(ctx, client, scope.Region, idStr)
		results = append(results, resource.NukeResult{Identifier: idStr, Error: err})
	}

	return results
}

// removeGlobalClusterMemberships removes all members from a global cluster that are in the specified region.
func removeGlobalClusterMemberships(ctx context.Context, client DBGlobalClusterMembershipsAPI, region, globalClusterID string) error {
	// Get the global cluster details
	result, err := client.DescribeGlobalClusters(ctx, &rds.DescribeGlobalClustersInput{
		GlobalClusterIdentifier: aws.String(globalClusterID),
	})
	if err != nil {
		return fmt.Errorf("failed to describe global cluster: %w", err)
	}

	if len(result.GlobalClusters) != 1 {
		return fmt.Errorf("expected 1 global cluster, got %d", len(result.GlobalClusters))
	}

	cluster := result.GlobalClusters[0]
	var removedMembers []string

	for _, member := range cluster.GlobalClusterMembers {
		memberRegion := extractRegionFromARN(aws.ToString(member.DBClusterArn))

		// Only remove members in the current region
		if region != "" && region != memberRegion {
			logging.Debugf("Skipping member %s (region %s != %s)", aws.ToString(member.DBClusterArn), memberRegion, region)
			continue
		}

		logging.Debugf("Removing cluster %s from global cluster %s", aws.ToString(member.DBClusterArn), globalClusterID)
		_, err := client.RemoveFromGlobalCluster(ctx, &rds.RemoveFromGlobalClusterInput{
			GlobalClusterIdentifier: cluster.GlobalClusterIdentifier,
			DbClusterIdentifier:     member.DBClusterArn,
		})
		if err != nil {
			return fmt.Errorf("failed to remove cluster %s from global cluster: %w", aws.ToString(member.DBClusterArn), err)
		}

		removedMembers = append(removedMembers, aws.ToString(member.DBClusterArn))
	}

	// Wait for all removed members to be fully detached
	for _, memberARN := range removedMembers {
		if err := waitForMemberRemoval(ctx, client, globalClusterID, memberARN); err != nil {
			return err
		}
	}

	if len(removedMembers) > 0 {
		logging.Debugf("Removed %d members from global cluster %s", len(removedMembers), globalClusterID)
	}

	return nil
}

// extractRegionFromARN extracts the region from an ARN string.
// ARN format: arn:aws:rds:REGION:account:cluster:name
func extractRegionFromARN(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

// waitForMemberRemoval waits for a cluster to be removed from the global cluster.
func waitForMemberRemoval(ctx context.Context, client DBGlobalClusterMembershipsAPI, globalClusterID, memberARN string) error {
	const (
		retryDelay = 10 * time.Second
		maxRetries = 90 // up to 15 minutes
	)

	for i := 0; i < maxRetries; i++ {
		result, err := client.DescribeGlobalClusters(ctx, &rds.DescribeGlobalClustersInput{
			GlobalClusterIdentifier: aws.String(globalClusterID),
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}

		// Check if the member is still present
		memberFound := false
		for _, cluster := range result.GlobalClusters {
			for _, member := range cluster.GlobalClusterMembers {
				if aws.ToString(member.DBClusterArn) == memberARN {
					memberFound = true
					break
				}
			}
		}

		if !memberFound {
			return nil // Member successfully removed
		}

		logging.Debugf("Waiting for cluster %s to be removed from global cluster", memberARN)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("cluster %s was not removed from global cluster within timeout", memberARN)
}
