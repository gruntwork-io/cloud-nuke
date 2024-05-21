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

type VPCLatticeServiceNetwork struct {
	BaseAwsResource
	Client vpclatticeiface.VPCLatticeAPI
	Region string
	ARNs   []string
}

func (n *VPCLatticeServiceNetwork) Init(session *session.Session) {
	n.Client = vpclattice.New(session)
}

// ResourceName - the simple name of the aws resource
func (n *VPCLatticeServiceNetwork) ResourceName() string {
	return "vpc-lattice-service-network"
}

// ResourceIdentifiers - the arns of the aws certificate manager certificates
func (n *VPCLatticeServiceNetwork) ResourceIdentifiers() []string {
	return n.ARNs
}

func (n *VPCLatticeServiceNetwork) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.VPCLatticeServiceNetwork
}

func (n *VPCLatticeServiceNetwork) ResourceServiceName() string {
	return "VPC Lattice Service Network"
}

func (n *VPCLatticeServiceNetwork) MaxBatchSize() int {
	return maxBatchSize
}

func (n *VPCLatticeServiceNetwork) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := n.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	n.ARNs = awsgo.StringValueSlice(identifiers)
	return n.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (n *VPCLatticeServiceNetwork) Nuke(arns []string) error {
	if err := n.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
