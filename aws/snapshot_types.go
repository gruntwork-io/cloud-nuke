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

// Nuke - nuke 'em all!!!
func (snapshot Snapshots) Nuke(session *session.Session) error {
	if err := nukeAllSnapshots(session, awsgo.StringSlice(snapshot.SnapshotIds)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
