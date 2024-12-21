package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type AMIsAPI interface {
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

// AMIs - represents all user owned AMIs
type AMIs struct {
	BaseAwsResource
	Client   AMIsAPI
	Region   string
	ImageIds []string
}

func (ami *AMIs) InitV2(cfg aws.Config) {
	ami.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ami *AMIs) ResourceName() string {
	return "ami"
}

// ResourceIdentifiers - The AMI image ids
func (ami *AMIs) ResourceIdentifiers() []string {
	return ami.ImageIds
}

func (ami *AMIs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ami *AMIs) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.AMI
}

func (ami *AMIs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ami.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ami.ImageIds = aws.ToStringSlice(identifiers)
	return ami.ImageIds, nil
}

// Nuke - nuke 'em all!!!
func (ami *AMIs) Nuke(identifiers []string) error {
	if err := ami.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ImageAvailableError struct{}

func (e ImageAvailableError) Error() string {
	return "Image didn't become available within wait attempts"
}
