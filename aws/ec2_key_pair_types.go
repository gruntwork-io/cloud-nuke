package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2KeyPairs struct {
	Client     ec2iface.EC2API
	Region     string
	KeyPairIds []string
}

// ResourceName - the simple name of the aws resource
func (k EC2KeyPairs) ResourceName() string {
	return "ec2-keypairs"
}

// ResourceIdentifiers - IDs of the ec2 key pairs
func (k EC2KeyPairs) ResourceIdentifiers() []string {
	return k.KeyPairIds
}

func (k EC2KeyPairs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

func (k EC2KeyPairs) Nuke(session *session.Session, identifiers []string) error {
	if err := k.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
