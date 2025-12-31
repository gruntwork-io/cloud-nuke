package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EgressOnlyIGAPI interface {
	DescribeEgressOnlyInternetGateways(ctx context.Context, params *ec2.DescribeEgressOnlyInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeEgressOnlyInternetGatewaysOutput, error)
	DeleteEgressOnlyInternetGateway(ctx context.Context, params *ec2.DeleteEgressOnlyInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteEgressOnlyInternetGatewayOutput, error)
}

// EgressOnlyInternetGateway represents all Egress only internet gateway
type EgressOnlyInternetGateway struct {
	BaseAwsResource
	Client EgressOnlyIGAPI
	Region string
	Pools  []string
}

func (egigw *EgressOnlyInternetGateway) Init(cfg aws.Config) {
	egigw.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (egigw *EgressOnlyInternetGateway) ResourceName() string {
	return "egress-only-internet-gateway"
}

func (egigw *EgressOnlyInternetGateway) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the Egress only igw
func (egigw *EgressOnlyInternetGateway) ResourceIdentifiers() []string {
	return egigw.Pools
}

func (egigw *EgressOnlyInternetGateway) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EgressOnlyInternetGateway
}

func (egigw *EgressOnlyInternetGateway) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := egigw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	egigw.Pools = aws.ToStringSlice(identifiers)
	return egigw.Pools, nil
}

// Nuke - nuke 'em all!!!
func (egigw *EgressOnlyInternetGateway) Nuke(ctx context.Context, identifiers []string) error {
	if err := egigw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
