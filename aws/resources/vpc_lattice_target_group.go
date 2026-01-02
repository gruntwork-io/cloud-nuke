package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
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
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[VPCLatticeTargetGroupAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = vpclattice.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.VPCLatticeTargetGroup
		},
		Lister: listVPCLatticeTargetGroups,
		Nuker:  resource.MultiStepDeleter(deregisterVPCLatticeTargets, deleteVPCLatticeTargetGroup),
	})
}

// listVPCLatticeTargetGroups retrieves all VPC Lattice Target Groups that match the config filters.
func listVPCLatticeTargetGroups(ctx context.Context, client VPCLatticeTargetGroupAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string
	paginator := vpclattice.NewListTargetGroupsPaginator(client, &vpclattice.ListTargetGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, item := range page.Items {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: item.Name,
				Time: item.CreatedAt,
			}) {
				ids = append(ids, item.Arn)
			}
		}
	}

	return ids, nil
}

// deregisterVPCLatticeTargets deregisters all targets from a target group.
func deregisterVPCLatticeTargets(ctx context.Context, client VPCLatticeTargetGroupAPI, identifier *string) error {
	var targets []types.Target
	paginator := vpclattice.NewListTargetsPaginator(client, &vpclattice.ListTargetsInput{
		TargetGroupIdentifier: identifier,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[ListTargets Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		for _, target := range page.Items {
			targets = append(targets, types.Target{
				Id: target.Id,
			})
		}
	}

	if len(targets) == 0 {
		return nil
	}

	_, err := client.DeregisterTargets(ctx, &vpclattice.DeregisterTargetsInput{
		TargetGroupIdentifier: identifier,
		Targets:               targets,
	})
	if err != nil {
		logging.Debugf("[DeregisterTargets Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	return nil
}

// deleteVPCLatticeTargetGroup deletes a single VPC Lattice Target Group.
func deleteVPCLatticeTargetGroup(ctx context.Context, client VPCLatticeTargetGroupAPI, identifier *string) error {
	_, err := client.DeleteTargetGroup(ctx, &vpclattice.DeleteTargetGroupInput{
		TargetGroupIdentifier: identifier,
	})
	return errors.WithStackTrace(err)
}
