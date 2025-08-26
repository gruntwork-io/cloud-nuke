package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudMapNamespacesAPI interface {
	ListNamespaces(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error)
	DeleteNamespace(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error)
	GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error)
	ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error)
}

type CloudMapNamespaces struct {
	BaseAwsResource
	Client       CloudMapNamespacesAPI
	Region       string
	NamespaceIds []string
}

func (cns *CloudMapNamespaces) Init(cfg aws.Config) {
	cns.Client = servicediscovery.NewFromConfig(cfg)
}

func (cns *CloudMapNamespaces) ResourceName() string {
	return "cloudmap-namespace"
}

func (cns *CloudMapNamespaces) ResourceIdentifiers() []string {
	return cns.NamespaceIds
}

func (cns *CloudMapNamespaces) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudMapNamespace
}

func (cns *CloudMapNamespaces) MaxBatchSize() int {
	return 50
}

func (cns *CloudMapNamespaces) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cns.getAll(c, configObj)
	if err != nil {
		return nil, err
	}
	
	cns.NamespaceIds = aws.ToStringSlice(identifiers)
	return cns.NamespaceIds, nil
}

func (cns *CloudMapNamespaces) Nuke(identifiers []string) error {
	if err := cns.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}