package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type VPCLatticeServiceNetworkAPI interface {
	ListServiceNetworks(ctx context.Context, params *vpclattice.ListServiceNetworksInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworksOutput, error)
	DeleteServiceNetwork(ctx context.Context, params *vpclattice.DeleteServiceNetworkInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkOutput, error)
	ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error)
	DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error)
}

type VPCLatticeServiceNetwork struct {
	BaseAwsResource
	Client VPCLatticeServiceNetworkAPI
	Region string
	ARNs   []string
}

func (sch *VPCLatticeServiceNetwork) InitV2(cfg awsgo.Config) {
	sch.Client = vpclattice.NewFromConfig(cfg)
}

func (sch *VPCLatticeServiceNetwork) IsUsingV2() bool { return true }

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

	n.ARNs = awsgo.ToStringSlice(identifiers)
	return n.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (n *VPCLatticeServiceNetwork) Nuke(arns []string) error {
	if err := n.nukeAll(awsgo.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
