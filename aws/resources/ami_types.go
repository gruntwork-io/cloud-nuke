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

// AMIs - represents all user owned AMIs
type AMIs struct {
	BaseAwsResource
	Client   ec2iface.EC2API
	Region   string
	ImageIds []string
}

func (ami *AMIs) Init(session *session.Session) {
	ami.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (ami *AMIs) ResourceName() string {
	return "ami"
}

// ResourceIdentifiers - The AMI image ids
func (ami *AMIs) ResourceIdentifiers() []string {
	return ami.ImageIds
}

func (ami *AMIs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ami *AMIs) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.AMI
}

func (ami *AMIs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ami.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ami.ImageIds = awsgo.StringValueSlice(identifiers)
	return ami.ImageIds, nil
}

// Nuke - nuke 'em all!!!
func (ami *AMIs) Nuke(identifiers []string) error {
	if err := ami.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ImageAvailableError struct{}

func (e ImageAvailableError) Error() string {
	return "Image didn't become available within wait attempts"
}
