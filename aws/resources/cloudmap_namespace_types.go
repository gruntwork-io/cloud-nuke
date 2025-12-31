package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudMapNamespacesAPI defines the interface for the AWS Cloud Map API operations needed to manage namespaces.
// This interface is used for both real AWS SDK clients and mock implementations in tests.
type CloudMapNamespacesAPI interface {
	ListNamespaces(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error)
	DeleteNamespace(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error)
	GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error)
	ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error)
	ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error)
}

// CloudMapNamespaces represents all Cloud Map namespaces found in a region.
// It embeds BaseAwsResource to inherit common resource management functionality.
type CloudMapNamespaces struct {
	BaseAwsResource
	Client       CloudMapNamespacesAPI
	Region       string
	NamespaceIds []string // Collection of namespace IDs to be managed
}

// Init initializes the CloudMapNamespaces resource with an AWS configuration.
// It creates the AWS Cloud Map client from the provided configuration.
func (cns *CloudMapNamespaces) Init(cfg aws.Config) {
	cns.Client = servicediscovery.NewFromConfig(cfg)
}

// ResourceName returns the descriptive name of this resource type.
// This name is used in logging and reporting.
func (cns *CloudMapNamespaces) ResourceName() string {
	return "cloudmap-namespace"
}

func (cns *CloudMapNamespaces) ResourceIdentifiers() []string {
	return cns.NamespaceIds
}

func (cns *CloudMapNamespaces) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudMapNamespace
}

// MaxBatchSize returns the maximum number of resources that should be processed in a single batch.
// This helps prevent API rate limiting and timeouts.
func (cns *CloudMapNamespaces) MaxBatchSize() int {
	return 50
}

// GetAndSetIdentifiers retrieves all Cloud Map namespace IDs that match the given configuration.
// It stores the found IDs internally and returns them.
func (cns *CloudMapNamespaces) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cns.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cns.NamespaceIds = aws.ToStringSlice(identifiers)
	return cns.NamespaceIds, nil
}

// Nuke deletes all Cloud Map namespaces identified by the provided IDs.
// It ensures proper cleanup by waiting for dependent services to be deleted first.
func (cns *CloudMapNamespaces) Nuke(ctx context.Context, identifiers []string) error {
	if err := cns.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
