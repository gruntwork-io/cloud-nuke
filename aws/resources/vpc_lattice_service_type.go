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

type VPCLatticeService struct {
	BaseAwsResource
	Client vpclatticeiface.VPCLatticeAPI
	Region string
	ARNs   []string
}

func (n *VPCLatticeService) Init(session *session.Session) {
	n.Client = vpclattice.New(session)
}

// ResourceName - the simple name of the aws resource
func (n *VPCLatticeService) ResourceName() string {
	return "vpc-lattice-service"
}

// ResourceIdentifiers - the arns of the aws certificate manager certificates
func (n *VPCLatticeService) ResourceIdentifiers() []string {
	return n.ARNs
}

func (n *VPCLatticeService) ResourceServiceName() string {
	return "VPC Lattice Service"
}

func (n *VPCLatticeService) MaxBatchSize() int {
	return maxBatchSize
}

func (n *VPCLatticeService) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.VPCLatticeService
}

func (n *VPCLatticeService) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := n.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	n.ARNs = awsgo.StringValueSlice(identifiers)
	return n.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (n *VPCLatticeService) Nuke(arns []string) error {
	if err := n.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
