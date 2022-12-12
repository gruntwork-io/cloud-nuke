package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2Instances - represents all ec2 instances
type EC2Instances struct {
	InstanceIds []string
}

// ResourceName - the simple name of the aws resource
func (instance EC2Instances) ResourceName() string {
	return "ec2"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (instance EC2Instances) ResourceIdentifiers() []string {
	return instance.InstanceIds
}

func (instance EC2Instances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (instance EC2Instances) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEc2Instances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type EC2VPCs struct {
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
