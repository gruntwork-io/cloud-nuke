package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ebs/ebsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// EBSVolumes - represents all ebs volumes
type EBSVolumes struct {
	Client    ebsiface.EBSAPI
	Region    string
	VolumeIds []string
}

// ResourceName - the simple name of the aws resource
func (volume EBSVolumes) ResourceName() string {
	return "ebs"
}

// ResourceIdentifiers - The volume ids of the ebs volumes
func (volume EBSVolumes) ResourceIdentifiers() []string {
	return volume.VolumeIds
}

func (volume EBSVolumes) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (volume EBSVolumes) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEbsVolumes(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
