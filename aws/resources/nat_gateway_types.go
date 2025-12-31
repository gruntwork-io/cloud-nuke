package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NatGatewaysAPI interface {
	DeleteNatGateway(ctx context.Context, params *ec2.DeleteNatGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNatGatewayOutput, error)
	DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)
}

// NatGateways - represents all AWS secrets manager secrets that should be deleted.
type NatGateways struct {
	BaseAwsResource
	Client        NatGatewaysAPI
	Region        string
	NatGatewayIDs []string
}

func (ngw *NatGateways) Init(cfg aws.Config) {
	ngw.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ngw *NatGateways) ResourceName() string {
	return "nat-gateway"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ngw *NatGateways) ResourceIdentifiers() []string {
	return ngw.NatGatewayIDs
}

func (ngw *NatGateways) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (ngw *NatGateways) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NatGateway
}

func (ngw *NatGateways) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ngw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ngw.NatGatewayIDs = aws.ToStringSlice(identifiers)
	return ngw.NatGatewayIDs, nil
}

// Nuke - nuke 'em all!!!
func (ngw *NatGateways) Nuke(ctx context.Context, identifiers []string) error {
	if err := ngw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
