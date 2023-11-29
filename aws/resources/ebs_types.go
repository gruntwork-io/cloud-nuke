package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumes - represents all ebs volumes
type EBSVolumes struct {
	Client    ec2iface.EC2API
	Region    string
	VolumeIds []string
}

func (ev *EBSVolumes) Init(session *session.Session) {
	ev.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (ev *EBSVolumes) ResourceName() string {
	return "ebs"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ev *EBSVolumes) ResourceIdentifiers() []string {
	return ev.VolumeIds
}

func (ev *EBSVolumes) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ev *EBSVolumes) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ev.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ev.VolumeIds = awsgo.StringValueSlice(identifiers)
	return ev.VolumeIds, nil
}

// Nuke - nuke 'em all!!!
func (ev *EBSVolumes) Nuke(identifiers []string) error {
	if err := ev.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
