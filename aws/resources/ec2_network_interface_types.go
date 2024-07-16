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

type NetworkInterface struct {
	BaseAwsResource
	Client       ec2iface.EC2API
	Region       string
	InterfaceIds []string
}

func (ni *NetworkInterface) Init(session *session.Session) {
	ni.BaseAwsResource.Init(session)
	ni.Client = ec2.New(session)
}

func (ni *NetworkInterface) ResourceName() string {
	return "network-interface"
}

func (ni *NetworkInterface) ResourceIdentifiers() []string {
	return ni.InterfaceIds
}

func (ni *NetworkInterface) MaxBatchSize() int {
	return 50
}

func (ni *NetworkInterface) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.NetworkInterface
}

func (ni *NetworkInterface) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ni.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ni.InterfaceIds = aws.StringValueSlice(identifiers)
	return ni.InterfaceIds, nil
}

func (ni *NetworkInterface) Nuke(identifiers []string) error {
	if err := ni.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
