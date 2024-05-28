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

// IPAM Pool- represents all IPAMs
type EC2IPAMPool struct {
	BaseAwsResource
	Client ec2iface.EC2API
	Region string
	Pools  []string
}

func (pool *EC2IPAMPool) Init(session *session.Session) {
	pool.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (pool *EC2IPAMPool) ResourceName() string {
	return "ipam-pool"
}

func (pool *EC2IPAMPool) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (pool *EC2IPAMPool) ResourceIdentifiers() []string {
	return pool.Pools
}

func (pool *EC2IPAMPool) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2IPAMPool
}

func (pool *EC2IPAMPool) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pool.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pool.Pools = awsgo.StringValueSlice(identifiers)
	return pool.Pools, nil
}

// Nuke - nuke 'em all!!!
func (pool *EC2IPAMPool) Nuke(identifiers []string) error {
	if err := pool.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
