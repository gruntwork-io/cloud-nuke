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

// Ec2Endpoint - represents all ec2 endpoints
type EC2Endpoints struct {
	BaseAwsResource
	Client    ec2iface.EC2API
	Region    string
	Endpoints []string
}

func (e *EC2Endpoints) Init(session *session.Session) {
	e.BaseAwsResource.Init(session)
	e.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (e *EC2Endpoints) ResourceName() string {
	return "ec2-endpoint"
}

func (e *EC2Endpoints) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers
func (e *EC2Endpoints) ResourceIdentifiers() []string {
	return e.Endpoints
}

func (e *EC2Endpoints) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := e.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	e.Endpoints = awsgo.StringValueSlice(identifiers)
	return e.Endpoints, nil
}

// Nuke - nuke 'em all!!!
func (e *EC2Endpoints) Nuke(identifiers []string) error {
	if err := e.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
