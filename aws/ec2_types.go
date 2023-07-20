package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2Instance - represents all ec2 instances
type EC2Instance struct {
	Client      ec2iface.EC2API
	Region      string
	InstanceIds []string
}

// ResourceName - the simple name of the aws resource
func (instance EC2Instance) ResourceName() string {
	return "ec2-instance"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (instance EC2Instance) ResourceIdentifiers() []string {
	return instance.InstanceIds
}

func (instance EC2Instance) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance EC2Instance) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEc2Instances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
