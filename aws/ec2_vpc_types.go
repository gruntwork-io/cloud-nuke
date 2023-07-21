package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2VPCs struct {
	Client ec2iface.EC2API
	Region string
	VPCIds []string
	VPCs   []Vpc
}

// ResourceName - the simple name of the aws resource
func (v EC2VPCs) ResourceName() string {
	return "vpc"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (v EC2VPCs) ResourceIdentifiers() []string {
	return v.VPCIds
}

func (v EC2VPCs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (v EC2VPCs) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllVPCs(session, identifiers, v.VPCs); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
