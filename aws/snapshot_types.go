package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Snapshots - represents all user owned Snapshots
type Snapshots struct {
	SnapshotIds []string
}

// ResourceName - the simple name of the aws resource
func (snapshot Snapshots) ResourceName() string {
	return "snap"
}

// ResourceIdentifiers - The Snapshot snapshot ids
func (snapshot Snapshots) ResourceIdentifiers() []string {
	return snapshot.SnapshotIds
}

func (snapshot Snapshots) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (snapshot Snapshots) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllSnapshots(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
