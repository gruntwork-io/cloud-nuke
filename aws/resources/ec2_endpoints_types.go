package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2EndpointsAPI interface {
	DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error)
}

// EC2Endpoints - represents all ec2 endpoints
type EC2Endpoints struct {
	BaseAwsResource
	Client    EC2EndpointsAPI
	Region    string
	Endpoints []string
}

func (e *EC2Endpoints) Init(cfg aws.Config) {
	e.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (e *EC2Endpoints) ResourceName() string {
	return "ec2-endpoint"
}

func (e *EC2Endpoints) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (e *EC2Endpoints) ResourceIdentifiers() []string {
	return e.Endpoints
}

func (e *EC2Endpoints) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2Endpoint
}

func (e *EC2Endpoints) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := e.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	e.Endpoints = aws.ToStringSlice(identifiers)
	return e.Endpoints, nil
}

// Nuke - nuke 'em all!!!
func (e *EC2Endpoints) Nuke(ctx context.Context, identifiers []string) error {
	if err := e.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
