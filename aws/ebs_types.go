package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumes - represents all ebs volumes
type EBSVolumes struct {
	Client    ec2iface.EC2API
	Region    string
	VolumeIds []string
}

// ResourceName - the simple name of the aws resource
func (ev EBSVolumes) ResourceName() string {
	return "ebs"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (ev EBSVolumes) ResourceIdentifiers() []string {
	return ev.VolumeIds
}

func (ev EBSVolumes) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (ev EBSVolumes) Nuke(session *session.Session, identifiers []string) error {
	if err := ev.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
