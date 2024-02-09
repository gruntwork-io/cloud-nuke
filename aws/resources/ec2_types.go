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

// EC2Instances - represents all ec2 instances
type EC2Instances struct {
	BaseAwsResource
	Client      ec2iface.EC2API
	Region      string
	InstanceIds []string
}

func (ei *EC2Instances) Init(session *session.Session) {
	ei.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (ei *EC2Instances) ResourceName() string {
	return "ec2"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ei *EC2Instances) ResourceIdentifiers() []string {
	return ei.InstanceIds
}

func (ei *EC2Instances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ei *EC2Instances) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ei.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ei.InstanceIds = awsgo.StringValueSlice(identifiers)
	return ei.InstanceIds, nil
}

// Nuke - nuke 'em all!!!
func (ei *EC2Instances) Nuke(identifiers []string) error {
	if err := ei.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
