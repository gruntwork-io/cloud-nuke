package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/vpclattice"
	"github.com/aws/aws-sdk-go/service/vpclattice/vpclatticeiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type VPCLatticeTargetGroup struct {
	BaseAwsResource
	Client       vpclatticeiface.VPCLatticeAPI
	Region       string
	ARNs         []string
	TargetGroups map[string]*vpclattice.TargetGroupSummary
}

func (n *VPCLatticeTargetGroup) Init(session *session.Session) {
	n.Client = vpclattice.New(session)
	n.TargetGroups = make(map[string]*vpclattice.TargetGroupSummary)
}

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

	n.ARNs = awsgo.StringValueSlice(identifiers)
	return n.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (n *VPCLatticeTargetGroup) Nuke(arns []string) error {
	if err := n.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
