package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type VPCLatticeServiceAPI interface {
	ListServices(ctx context.Context, params *vpclattice.ListServicesInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServicesOutput, error)
	DeleteService(ctx context.Context, params *vpclattice.DeleteServiceInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceOutput, error)
	ListServiceNetworkServiceAssociations(ctx context.Context, params *vpclattice.ListServiceNetworkServiceAssociationsInput, optFns ...func(*vpclattice.Options)) (*vpclattice.ListServiceNetworkServiceAssociationsOutput, error)
	DeleteServiceNetworkServiceAssociation(ctx context.Context, params *vpclattice.DeleteServiceNetworkServiceAssociationInput, optFns ...func(*vpclattice.Options)) (*vpclattice.DeleteServiceNetworkServiceAssociationOutput, error)
}

type VPCLatticeService struct {
	BaseAwsResource
	Client VPCLatticeServiceAPI
	Region string
	ARNs   []string
}

func (sch *VPCLatticeService) InitV2(cfg aws.Config) {
	sch.Client = vpclattice.NewFromConfig(cfg)
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

	n.ARNs = aws.ToStringSlice(identifiers)
	return n.ARNs, nil
}

// Nuke - nuke 'em all!!!
func (n *VPCLatticeService) Nuke(arns []string) error {
	if err := n.nukeAll(aws.StringSlice(arns)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
