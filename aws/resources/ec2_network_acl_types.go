package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NetworkACLAPI interface {
	DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)
	DeleteNetworkAcl(ctx context.Context, params *ec2.DeleteNetworkAclInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkAclOutput, error)
	ReplaceNetworkAclAssociation(ctx context.Context, params *ec2.ReplaceNetworkAclAssociationInput, optFns ...func(*ec2.Options)) (*ec2.ReplaceNetworkAclAssociationOutput, error) // Add this line
}

type NetworkACL struct {
	BaseAwsResource
	Client NetworkACLAPI
	Region string
	Ids    []string
}

func (nacl *NetworkACL) Init(cfg aws.Config) {
	nacl.Client = ec2.NewFromConfig(cfg)
}

func (nacl *NetworkACL) ResourceName() string {
	return "network-acl"
}

func (nacl *NetworkACL) ResourceIdentifiers() []string {
	return nacl.Ids
}

func (nacl *NetworkACL) MaxBatchSize() int {
	return 50
}

func (nacl *NetworkACL) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkACL
}

func (nacl *NetworkACL) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nacl.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nacl.Ids = aws.ToStringSlice(identifiers)
	return nacl.Ids, nil
}

func (nacl *NetworkACL) Nuke(identifiers []string) error {
	if err := nacl.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
