package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// Snapshots - represents all user owned Snapshots
type Snapshots struct {
	Client      ec2iface.EC2API
	Region      string
	SnapshotIds []string
}

// ResourceName - the simple name of the aws resource
func (s Snapshots) ResourceName() string {
	return "snap"
}

// ResourceIdentifiers - The Snapshot snapshot ids
func (s Snapshots) ResourceIdentifiers() []string {
	return s.SnapshotIds
}

func (s Snapshots) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (s Snapshots) Nuke(session *session.Session, identifiers []string) error {
	if err := s.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
