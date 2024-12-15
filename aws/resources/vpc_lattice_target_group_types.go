package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type VPCLatticeAPI interface {
	ListTargetGroups(ctx context.Context, params *vpclattice.ListTargetGroupsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTargetGroupsOutput, error)
	ListTargets(ctx context.Context, params *vpclattice.ListTargetsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListTargetsOutput, error)
	DeregisterTargets(ctx context.Context, params *vpclattice.DeregisterTargetsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeregisterTargetsOutput, error)
	DeleteTargetGroup(ctx context.Context, params *vpclattice.DeleteTargetGroupInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteTargetGroupOutput, error)
}

type VPCLatticeTargetGroup struct {
	BaseAwsResource
	Client       VPCLatticeAPI
	Region       string
	ARNs         []string
	TargetGroups map[string]*types.TargetGroupSummary
}

func (sch *VPCLatticeTargetGroup) InitV2(cfg aws.Config) {
	sch.Client = vpclattice.NewFromConfig(cfg)
	sch.TargetGroups = make(map[string]*types.TargetGroupSummary, 0)
}

func (sch *VPCLatticeTargetGroup) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (n *VPCLatticeTargetGroup) ResourceName() string {
	return "vpc-lattice-target-group"
}

// ResourceIdentifiers - the arns of the aws certificate manager certificates
func (n *VPCLatticeTargetGroup) ResourceIdentifiers() []string {
	return n.ARNs
}

func (n *VPCLatticeTargetGroup) ResourceServiceName() string {
	return "VPC Lattice Target Group"
}

func (n *VPCLatticeTargetGroup) MaxBatchSize() int {
	return maxBatchSize
}

func (n *VPCLatticeTargetGroup) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.VPCLatticeTargetGroup
}

func (n *VPCLatticeTargetGroup) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := n.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	n.ARNs = aws.ToStringSlice(identifiers)
	return n.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (n *VPCLatticeTargetGroup) Nuke(arns []string) error {
	if err := n.nukeAll(aws.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
