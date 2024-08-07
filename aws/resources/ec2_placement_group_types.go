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

type EC2PlacementGroups struct {
	BaseAwsResource
	Client              ec2iface.EC2API
	Region              string
	PlacementGroupNames []string
}

func (k *EC2PlacementGroups) Init(session *session.Session) {
	k.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (k *EC2PlacementGroups) ResourceName() string {
	return "ec2-placement-groups"
}

// ResourceIdentifiers - IDs of the ec2 key pairs
func (k *EC2PlacementGroups) ResourceIdentifiers() []string {
	return k.PlacementGroupNames
}

func (k *EC2PlacementGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

func (k *EC2PlacementGroups) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2PlacementGroups
}

func (k *EC2PlacementGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := k.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	k.PlacementGroupNames = awsgo.StringValueSlice(identifiers)
	return k.PlacementGroupNames, nil
}

func (k *EC2PlacementGroups) Nuke(identifiers []string) error {
	if err := k.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
