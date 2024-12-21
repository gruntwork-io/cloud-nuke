package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2KeyPairsAPI interface {
	DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error)
	DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
}

type EC2KeyPairs struct {
	BaseAwsResource
	Client     EC2KeyPairsAPI
	Region     string
	KeyPairIds []string
}

func (k *EC2KeyPairs) InitV2(cfg aws.Config) {
	k.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (k *EC2KeyPairs) ResourceName() string {
	return "ec2-keypairs"
}

// ResourceIdentifiers - IDs of the ec2 key pairs
func (k *EC2KeyPairs) ResourceIdentifiers() []string {
	return k.KeyPairIds
}

func (k *EC2KeyPairs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

func (k *EC2KeyPairs) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2KeyPairs
}

func (k *EC2KeyPairs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := k.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	k.KeyPairIds = aws.ToStringSlice(identifiers)
	return k.KeyPairIds, nil
}

func (k *EC2KeyPairs) Nuke(identifiers []string) error {
	if err := k.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
