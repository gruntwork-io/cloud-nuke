package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LaunchTemplatesAPI interface {
	DescribeLaunchTemplates(ctx context.Context, params *ec2.DescribeLaunchTemplatesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeLaunchTemplatesOutput, error)
	DeleteLaunchTemplate(ctx context.Context, params *ec2.DeleteLaunchTemplateInput, optFns ...func(*ec2.Options)) (*ec2.DeleteLaunchTemplateOutput, error)
}

// LaunchTemplates - represents all launch templates
type LaunchTemplates struct {
	BaseAwsResource
	Client              LaunchTemplatesAPI
	Region              string
	LaunchTemplateNames []string
}

func (lt *LaunchTemplates) Init(cfg aws.Config) {
	lt.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (lt *LaunchTemplates) ResourceName() string {
	return "lt"
}

func (lt *LaunchTemplates) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The names of the launch templates
func (lt *LaunchTemplates) ResourceIdentifiers() []string {
	return lt.LaunchTemplateNames
}

func (lt *LaunchTemplates) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.LaunchTemplate
}

func (lt *LaunchTemplates) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := lt.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	lt.LaunchTemplateNames = aws.ToStringSlice(identifiers)
	return lt.LaunchTemplateNames, nil
}

// Nuke - nuke 'em all!!!
func (lt *LaunchTemplates) Nuke(ctx context.Context, identifiers []string) error {
	if err := lt.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
