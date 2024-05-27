package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// IPAM - represents all IPAMs
type EC2IPAMResourceDiscovery struct {
	BaseAwsResource
	Client       ec2iface.EC2API
	Region       string
	DiscoveryIDs []string
}

func (ipam *EC2IPAMResourceDiscovery) Init(session *session.Session) {
	ipam.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (ipam *EC2IPAMResourceDiscovery) ResourceName() string {
	return "ipam-resource-discovery"
}

func (ipam *EC2IPAMResourceDiscovery) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (ipam *EC2IPAMResourceDiscovery) ResourceIdentifiers() []string {
	return ipam.DiscoveryIDs
}

func (ipam *EC2IPAMResourceDiscovery) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2IPAMResourceDiscovery
}

func (ipam *EC2IPAMResourceDiscovery) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ipam.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ipam.DiscoveryIDs = awsgo.StringValueSlice(identifiers)
	return ipam.DiscoveryIDs, nil
}

// Nuke - nuke 'em all!!!
func (ipam *EC2IPAMResourceDiscovery) Nuke(identifiers []string) error {
	if err := ipam.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
