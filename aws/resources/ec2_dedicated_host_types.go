package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2DedicatedHostsAPI interface {
	DescribeHosts(ctx context.Context, params *ec2.DescribeHostsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeHostsOutput, error)
	ReleaseHosts(ctx context.Context, params *ec2.ReleaseHostsInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseHostsOutput, error)
}

// EC2DedicatedHosts - represents all host allocation IDs
type EC2DedicatedHosts struct {
	BaseAwsResource
	Client  EC2DedicatedHostsAPI
	Region  string
	HostIds []string
}

func (h *EC2DedicatedHosts) Init(cfg aws.Config) {
	h.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (h *EC2DedicatedHosts) ResourceName() string {
	return "ec2-dedicated-hosts"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (h *EC2DedicatedHosts) ResourceIdentifiers() []string {
	return h.HostIds
}

func (h *EC2DedicatedHosts) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (h *EC2DedicatedHosts) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2DedicatedHosts
}

func (h *EC2DedicatedHosts) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := h.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	h.HostIds = aws.ToStringSlice(identifiers)
	return h.HostIds, nil
}

// Nuke - nuke 'em all!!!
func (h *EC2DedicatedHosts) Nuke(identifiers []string) error {
	if err := h.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
