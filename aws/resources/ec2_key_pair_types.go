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

type EC2KeyPairs struct {
	BaseAwsResource
	Client     ec2iface.EC2API
	Region     string
	KeyPairIds []string
}

func (k *EC2KeyPairs) Init(session *session.Session) {
	k.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (k *EC2KeyPairs) ResourceName() string {
	return "ec2-keypairs"
}

// ResourceIdentifiers - IDs of the ec2 key pairs
func (k *EC2KeyPairs) ResourceIdentifiers() []string {
	return k.KeyPairIds
}

func (k *EC2KeyPairs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

func (k *EC2KeyPairs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := k.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	k.KeyPairIds = awsgo.StringValueSlice(identifiers)
	return k.KeyPairIds, nil
}

func (k *EC2KeyPairs) Nuke(identifiers []string) error {
	if err := k.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
