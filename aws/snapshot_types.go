package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// EC2Snapshot - represents all user owned EC2Snapshot
type EC2Snapshot struct {
	Client      ec2iface.EC2API
	Region      string
	SnapshotIds []string
}

// ResourceName - the simple name of the aws resource
func (snapshot EC2Snapshot) ResourceName() string {
	return "snap"
}

// ResourceIdentifiers - The EC2Snapshot snapshot ids
func (snapshot EC2Snapshot) ResourceIdentifiers() []string {
	return snapshot.SnapshotIds
}

func (snapshot EC2Snapshot) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (snapshot EC2Snapshot) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSnapshots(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
