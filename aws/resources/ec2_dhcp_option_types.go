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

type EC2DhcpOption struct {
	Client ec2iface.EC2API
	Region string
	VPCIds []string
}

func (v *EC2DhcpOption) Init(session *session.Session) {
	v.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (v *EC2DhcpOption) ResourceName() string {
	return "ec2_dhcp_option"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (v *EC2DhcpOption) ResourceIdentifiers() []string {
	return v.VPCIds
}

func (v *EC2DhcpOption) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (v *EC2DhcpOption) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := v.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	v.VPCIds = awsgo.StringValueSlice(identifiers)
	return v.VPCIds, nil
}

func (v *EC2DhcpOption) Nuke(identifiers []string) error {
	if err := v.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
