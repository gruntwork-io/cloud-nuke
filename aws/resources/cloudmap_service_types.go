package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudMapServicesAPI interface {
	ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error)
	DeleteService(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error)
	ListInstances(ctx context.Context, params *servicediscovery.ListInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListInstancesOutput, error)
	DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error)
}

type CloudMapServices struct {
	BaseAwsResource
	Client     CloudMapServicesAPI
	Region     string
	ServiceIds []string
}

func (cms *CloudMapServices) Init(cfg aws.Config) {
	cms.Client = servicediscovery.NewFromConfig(cfg)
}

func (cms *CloudMapServices) ResourceName() string {
	return "cloudmap-service"
}

func (cms *CloudMapServices) ResourceIdentifiers() []string {
	return cms.ServiceIds
}

func (cms *CloudMapServices) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudMapService
}

func (cms *CloudMapServices) MaxBatchSize() int {
	return 50
}

func (cms *CloudMapServices) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cms.getAll(c, configObj)
	if err != nil {
		return nil, err
	}
	
	cms.ServiceIds = aws.ToStringSlice(identifiers)
	return cms.ServiceIds, nil
}

func (cms *CloudMapServices) Nuke(identifiers []string) error {
	if err := cms.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}