package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2IPAMByoasnAPI interface {
	DescribeIpamByoasn(ctx context.Context, params *ec2.DescribeIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamByoasnOutput, error)
	DisassociateIpamByoasn(ctx context.Context, params *ec2.DisassociateIpamByoasnInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateIpamByoasnOutput, error)
}

// EC2IPAMByoasn IPAM Byoasn- represents all IPAMs
type EC2IPAMByoasn struct {
	BaseAwsResource
	Client EC2IPAMByoasnAPI
	Region string
	Pools  []string
}

var MaxResultCount = int32(10)

func (byoasn *EC2IPAMByoasn) Init(cfg aws.Config) {
	byoasn.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (byoasn *EC2IPAMByoasn) ResourceName() string {
	return "ipam-byoasn"
}

func (byoasn *EC2IPAMByoasn) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (byoasn *EC2IPAMByoasn) ResourceIdentifiers() []string {
	return byoasn.Pools
}

func (byoasn *EC2IPAMByoasn) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := byoasn.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	byoasn.Pools = aws.ToStringSlice(identifiers)
	return byoasn.Pools, nil
}

// Nuke - nuke 'em all!!!
func (byoasn *EC2IPAMByoasn) Nuke(identifiers []string) error {
	if err := byoasn.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
