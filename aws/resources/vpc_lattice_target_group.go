package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// VPCLatticeTargetGroupAPI defines the interface for VPC Lattice Target Group operations.
type VPCLatticeTargetGroupAPI interface {
	ListTargetGroups(ctx context.Context, params *vpclattice.ListTargetGroupsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTargetGroupsOutput, error)
	ListTargets(ctx context.Context, params *vpclattice.ListTargetsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTargetsOutput, error)
	DeregisterTargets(ctx context.Context, params *vpclattice.DeregisterTargetsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeregisterTargetsOutput, error)
	DeleteTargetGroup(ctx context.Context, params *vpclattice.DeleteTargetGroupInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteTargetGroupOutput, error)
}

// NewVPCLatticeTargetGroup creates a new VPC Lattice Target Group resource using the generic resource pattern.
func NewVPCLatticeTargetGroup() AwsResource {
	return NewAwsResource(&resource.Resource[VPCLatticeTargetGroupAPI]{
		ResourceTypeName: "vpc-lattice-target-group",
		BatchSize:        maxBatchSize,
		InitClient: func(r *resource.Resource[VPCLatticeTargetGroupAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for VPC Lattice client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = vpclattice.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.VPCLatticeTargetGroup
		},
		Lister: listVPCLatticeTargetGroups,
		Nuker:  deleteVPCLatticeTargetGroups,
	})
}

// listVPCLatticeTargetGroups retrieves all VPC Lattice Target Groups that match the config filters.
func listVPCLatticeTargetGroups(ctx context.Context, client VPCLatticeTargetGroupAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.ListTargetGroups(ctx, nil)
	if err != nil {
		return nil, err
	}

	var ids []*string
	for _, item := range output.Items {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: item.Name,
			Time: item.CreatedAt,
		}) {
			ids = append(ids, item.Arn)
		}
	}

	return ids, nil
}

// nukeVPCLatticeTargets deregisters all targets from a target group.
func nukeVPCLatticeTargets(ctx context.Context, client VPCLatticeTargetGroupAPI, identifier *string) error {
	output, err := client.ListTargets(ctx, &vpclattice.ListTargetsInput{
		TargetGroupIdentifier: identifier,
	})
	if err != nil {
		logging.Debugf("[ListTargets Failed] %s", err)
		return err
	}

	var targets []types.Target
	for _, target := range output.Items {
		targets = append(targets, types.Target{
			Id: target.Id,
		})
	}

	if len(targets) > 0 {
		_, err = client.DeregisterTargets(ctx, &vpclattice.DeregisterTargetsInput{
			TargetGroupIdentifier: identifier,
			Targets:               targets,
		})
		if err != nil {
			logging.Debugf("[DeregisterTargets Failed] %s", err)
			return err
		}
	}

	return nil
}

// deleteVPCLatticeTargetGroup deletes a single VPC Lattice Target Group after deregistering its targets.
func deleteVPCLatticeTargetGroup(ctx context.Context, client VPCLatticeTargetGroupAPI, identifier *string) error {
	// First deregister all targets
	if err := nukeVPCLatticeTargets(ctx, client, identifier); err != nil {
		return err
	}

	// Then delete the target group
	_, err := client.DeleteTargetGroup(ctx, &vpclattice.DeleteTargetGroupInput{
		TargetGroupIdentifier: identifier,
	})
	return err
}

// deleteVPCLatticeTargetGroups is a custom nuker for VPC Lattice Target Groups.
func deleteVPCLatticeTargetGroups(ctx context.Context, client VPCLatticeTargetGroupAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return nil
	}

	logging.Debugf("Deleting all %s in %s", resourceType, scope)

	deletedCount := 0
	for _, id := range identifiers {
		err := deleteVPCLatticeTargetGroup(ctx, client, id)

		report.Record(report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: resourceType,
			Error:        err,
		})

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedCount++
			logging.Debugf("Deleted %s: %s", resourceType, aws.ToString(id))
		}
	}

	logging.Debugf("[OK] %d %s(s) terminated in %s", deletedCount, resourceType, scope)
	return nil
}
