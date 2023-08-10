package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2VPCs struct {
	Client ec2iface.EC2API
	Region string
	VPCIds []string
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

func (v EC2VPCs) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := v.getAll(configObj)
	if err != nil {
		return nil, err
	}

	v.VPCIds = awsgo.StringValueSlice(identifiers)
	return v.VPCIds, nil
}

// Nuke - nuke 'em all!!!
func (v EC2VPCs) Nuke(identifiers []string) error {
	if err := v.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
