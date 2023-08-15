package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumes - represents all ebs volumes
type EIPAddresses struct {
	Client        ec2iface.EC2API
	Region        string
	AllocationIds []string
}

func (address *EIPAddresses) Init(session *session.Session) {
	address.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (address *EIPAddresses) ResourceName() string {
	return "eip"
}

// ResourceIdentifiers - The instance ids of the eip addresses
func (address *EIPAddresses) ResourceIdentifiers() []string {
	return address.AllocationIds
}

func (address *EIPAddresses) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (address *EIPAddresses) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := address.getAll(configObj)
	if err != nil {
		return nil, err
	}

	address.AllocationIds = awsgo.StringValueSlice(identifiers)
	return address.AllocationIds, nil
}

// Nuke - nuke 'em all!!!
func (address *EIPAddresses) Nuke(identifiers []string) error {
	if err := address.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
