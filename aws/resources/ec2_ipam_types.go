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
type EC2IPAMs struct {
	Client ec2iface.EC2API
	Region string
	IDs    []string
}

func (ipam *EC2IPAMs) Init(session *session.Session) {
	ipam.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (ipam *EC2IPAMs) ResourceName() string {
	return "ipam"
}

func (ipam *EC2IPAMs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (ipam *EC2IPAMs) ResourceIdentifiers() []string {
	return ipam.IDs
}

func (ipam *EC2IPAMs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ipam.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ipam.IDs = awsgo.StringValueSlice(identifiers)
	return ipam.IDs, nil
}

// Nuke - nuke 'em all!!!
func (ipam *EC2IPAMs) Nuke(identifiers []string) error {
	if err := ipam.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
