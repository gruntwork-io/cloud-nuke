package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DHCPOption struct {
	Id    *string
	VpcId *string
}

type EC2DhcpOptionAPI interface {
	AssociateDhcpOptions(ctx context.Context, params *ec2.AssociateDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.AssociateDhcpOptionsOutput, error)
	DescribeDhcpOptions(ctx context.Context, params *ec2.DescribeDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteDhcpOptions(ctx context.Context, params *ec2.DeleteDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteDhcpOptionsOutput, error)
}

type EC2DhcpOption struct {
	BaseAwsResource
	Client      EC2DhcpOptionAPI
	Region      string
	VPCIds      []string
	DhcpOptions map[string]DHCPOption
}

func (v *EC2DhcpOption) Init(cfg aws.Config) {
	v.Client = ec2.NewFromConfig(cfg)
	v.DhcpOptions = make(map[string]DHCPOption)
}

// ResourceName - the simple name of the aws resource
func (v *EC2DhcpOption) ResourceName() string {
	return "ec2_dhcp_option"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (v *EC2DhcpOption) ResourceIdentifiers() []string {
	return v.VPCIds
}

func (v *EC2DhcpOption) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (v *EC2DhcpOption) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := v.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	v.VPCIds = aws.ToStringSlice(identifiers)
	return v.VPCIds, nil
}

func (v *EC2DhcpOption) Nuke(identifiers []string) error {
	if err := v.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
