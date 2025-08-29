package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudMapServicesAPI defines the interface for the AWS Cloud Map API operations needed to manage services.
// This interface is used for both real AWS SDK clients and mock implementations in tests.
type CloudMapServicesAPI interface {
	ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error)
	DeleteService(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error)
	ListInstances(ctx context.Context, params *servicediscovery.ListInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListInstancesOutput, error)
	DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error)
}

// CloudMapServices represents all Cloud Map services found in a region.
// It embeds BaseAwsResource to inherit common resource management functionality.
type CloudMapServices struct {
	BaseAwsResource
	Client     CloudMapServicesAPI
	Region     string
	ServiceIds []string // Collection of service IDs to be managed
}

// Init initializes the CloudMapServices resource with an AWS configuration.
// It creates the AWS Cloud Map client from the provided configuration.
func (cms *CloudMapServices) Init(cfg aws.Config) {
	cms.Client = servicediscovery.NewFromConfig(cfg)
}

// ResourceName returns the descriptive name of this resource type.
// This name is used in logging and reporting.
func (cms *CloudMapServices) ResourceName() string {
	return "cloudmap-service"
}

func (cms *CloudMapServices) ResourceIdentifiers() []string {
	return cms.ServiceIds
}

func (cms *CloudMapServices) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudMapService
}

// MaxBatchSize returns the maximum number of resources that should be processed in a single batch.
// This helps prevent API rate limiting and timeouts.
func (cms *CloudMapServices) MaxBatchSize() int {
	return 50
}

// GetAndSetIdentifiers retrieves all Cloud Map service IDs that match the given configuration.
// It stores the found IDs internally and returns them.
func (cms *CloudMapServices) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cms.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cms.ServiceIds = aws.ToStringSlice(identifiers)
	return cms.ServiceIds, nil
}

// Nuke deletes all Cloud Map services identified by the provided IDs.
// It ensures proper cleanup by deregistering all service instances first.
func (cms *CloudMapServices) Nuke(identifiers []string) error {
	if err := cms.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
