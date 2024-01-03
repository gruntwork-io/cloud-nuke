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

// IPAM Byoasn- represents all IPAMs
type EC2IPAMCustomAllocation struct {
	Client               ec2iface.EC2API
	Region               string
	Allocations          []string
	PoolAndAllocationMap map[string]string
}

func (cs *EC2IPAMCustomAllocation) Init(session *session.Session) {
	cs.Client = ec2.New(session)
	cs.PoolAndAllocationMap = make(map[string]string)
}

// ResourceName - the simple name of the aws resource
func (cs *EC2IPAMCustomAllocation) ResourceName() string {
	return "ipam-custom-allocation"
}

func (cs *EC2IPAMCustomAllocation) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 1000
}

// ResourceIdentifiers - The ids of the IPAMs
func (cs *EC2IPAMCustomAllocation) ResourceIdentifiers() []string {
	return cs.Allocations
}

func (cs *EC2IPAMCustomAllocation) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cs.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cs.Allocations = awsgo.StringValueSlice(identifiers)
	return cs.Allocations, nil
}

// Nuke - nuke 'em all!!!
func (cs *EC2IPAMCustomAllocation) Nuke(identifiers []string) error {
	if err := cs.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
