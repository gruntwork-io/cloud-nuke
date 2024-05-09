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

// Ec2Subnet- represents all Subnets
type EC2Subnet struct {
	BaseAwsResource
	Client  ec2iface.EC2API
	Region  string
	Subnets []string
}

func (es *EC2Subnet) Init(session *session.Session) {
	es.BaseAwsResource.Init(session)
	es.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (es *EC2Subnet) ResourceName() string {
	return "ec2-subnet"
}

func (es *EC2Subnet) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the subnets
func (es *EC2Subnet) ResourceIdentifiers() []string {
	return es.Subnets
}

// func (es *EC2Subnet) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
// 	return configObj.EC2Subnet
// }

func (es *EC2Subnet) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := es.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	es.Subnets = awsgo.StringValueSlice(identifiers)
	return es.Subnets, nil
}

// Nuke - nuke 'em all!!!
func (es *EC2Subnet) Nuke(identifiers []string) error {
	if err := es.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
