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
type EC2IPAMByoasn struct {
	Client ec2iface.EC2API
	Region string
	Pools  []string
}

var MaxResultCount = int64(10)

func (byoasn *EC2IPAMByoasn) Init(session *session.Session) {
	byoasn.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (byoasn *EC2IPAMByoasn) ResourceName() string {
	return "ipam-byoasn"
}

func (byoasn *EC2IPAMByoasn) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (byoasn *EC2IPAMByoasn) ResourceIdentifiers() []string {
	return byoasn.Pools
}

func (byoasn *EC2IPAMByoasn) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := byoasn.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	byoasn.Pools = awsgo.StringValueSlice(identifiers)
	return byoasn.Pools, nil
}

// Nuke - nuke 'em all!!!
func (byoasn *EC2IPAMByoasn) Nuke(identifiers []string) error {
	if err := byoasn.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
