package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DBClusterSnapshots struct {
	SnapShots []string
}

// Name of the AWS resource
func (snapshot DBClusterSnapshots) ResourceName() string {
	return "rds-snapshots"
}

// Snapshot names of the RDS Cluster Snapshots
func (snapshot DBClusterSnapshots) ResourceIdentifiers() []string {
	return snapshot.SnapShots
}

// MaxBatchSize decides how many snapshots to delete in one call.
func (snapshot DBClusterSnapshots) MaxBatchSize() int {
	return 200
}

// Nuke/Delete all snapshots
func (snapshot DBClusterSnapshots) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllRdsClusterSnapshots(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
