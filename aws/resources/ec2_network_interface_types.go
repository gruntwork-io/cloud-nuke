package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	NetworkInterfaceTypeInterface = "interface"
)

type NetworkInterfaceAPI interface {
	DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}
type NetworkInterface struct {
	BaseAwsResource
	Client       NetworkInterfaceAPI
	Region       string
	InterfaceIds []string
}

func (ni *NetworkInterface) InitV2(cfg aws.Config) {
	ni.Client = ec2.NewFromConfig(cfg)
}

func (ni *NetworkInterface) ResourceName() string {
	return "network-interface"
}

func (ni *NetworkInterface) ResourceIdentifiers() []string {
	return ni.InterfaceIds
}

func (ni *NetworkInterface) MaxBatchSize() int {
	return 50
}

func (ni *NetworkInterface) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkInterface
}

func (ni *NetworkInterface) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ni.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ni.InterfaceIds = aws.ToStringSlice(identifiers)
	return ni.InterfaceIds, nil
}

func (ni *NetworkInterface) Nuke(identifiers []string) error {
	if err := ni.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
