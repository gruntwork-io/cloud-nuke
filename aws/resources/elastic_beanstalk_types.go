package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EBApplicationsAPI interface {
	DescribeApplications(ctx context.Context, params *elasticbeanstalk.DescribeApplicationsInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DescribeApplicationsOutput, error)
	DeleteApplication(ctx context.Context, params *elasticbeanstalk.DeleteApplicationInput, optFns ...func(*elasticbeanstalk.Options)) (*elasticbeanstalk.DeleteApplicationOutput, error)
}

// EBApplications - represents all elastic beanstalk applications
type EBApplications struct {
	BaseAwsResource
	Client EBApplicationsAPI
	Region string
	appIds []string
}

func (eb *EBApplications) InitV2(cfg aws.Config) {
	eb.Client = elasticbeanstalk.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (eb *EBApplications) ResourceName() string {
	return "elastic-beanstalk"
}

// ResourceIdentifiers - The application ids of the elastic beanstalk
func (eb *EBApplications) ResourceIdentifiers() []string {
	return eb.appIds
}

func (eb *EBApplications) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (eb *EBApplications) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ElasticBeanstalk
}

func (eb *EBApplications) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := eb.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	eb.appIds = aws.ToStringSlice(identifiers)
	return eb.appIds, nil
}

// Nuke - nuke 'em all!!!
func (eb *EBApplications) Nuke(identifiers []string) error {
	if err := eb.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
