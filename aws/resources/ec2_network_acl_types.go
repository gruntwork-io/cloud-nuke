package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type NetworkACL struct {
	BaseAwsResource
	Client ec2iface.EC2API
	Region string
	Ids    []string
}

func (nacl *NetworkACL) Init(session *session.Session) {
	nacl.BaseAwsResource.Init(session)
	nacl.Client = ec2.New(session)
}

func (nacl *NetworkACL) ResourceName() string {
	return "network-acl"
}

func (nacl *NetworkACL) ResourceIdentifiers() []string {
	return nacl.Ids
}

func (nacl *NetworkACL) MaxBatchSize() int {
	return 50
}

func (nacl *NetworkACL) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := nacl.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	nacl.Ids = aws.StringValueSlice(identifiers)
	return nacl.Ids, nil
}

func (nacl *NetworkACL) Nuke(identifiers []string) error {
	if err := nacl.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
