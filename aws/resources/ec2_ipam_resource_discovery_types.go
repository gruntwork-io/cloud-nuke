package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2IPAMResourceDiscoveryAPI interface {
	DescribeIpamResourceDiscoveries(ctx context.Context, params *ec2.DescribeIpamResourceDiscoveriesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamResourceDiscoveriesOutput, error)
	DeleteIpamResourceDiscovery(ctx context.Context, params *ec2.DeleteIpamResourceDiscoveryInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamResourceDiscoveryOutput, error)
}

// EC2IPAMResourceDiscovery IPAM - represents all IPAMs
type EC2IPAMResourceDiscovery struct {
	BaseAwsResource
	Client       EC2IPAMResourceDiscoveryAPI
	Region       string
	DiscoveryIDs []string
}

func (ipam *EC2IPAMResourceDiscovery) InitV2(cfg aws.Config) {
	ipam.Client = ec2.NewFromConfig(cfg)
}

func (ipam *EC2IPAMResourceDiscovery) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (ipam *EC2IPAMResourceDiscovery) ResourceName() string {
	return "ipam-resource-discovery"
}

func (ipam *EC2IPAMResourceDiscovery) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (ipam *EC2IPAMResourceDiscovery) ResourceIdentifiers() []string {
	return ipam.DiscoveryIDs
}

func (ipam *EC2IPAMResourceDiscovery) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2IPAMResourceDiscovery
}

func (ipam *EC2IPAMResourceDiscovery) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ipam.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ipam.DiscoveryIDs = aws.ToStringSlice(identifiers)
	return ipam.DiscoveryIDs, nil
}

// Nuke - nuke 'em all!!!
func (ipam *EC2IPAMResourceDiscovery) Nuke(identifiers []string) error {
	if err := ipam.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
