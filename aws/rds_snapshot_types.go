package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

type DBSnapshots struct {
	SnapShots []string
}

// Name of the AWS resource
func (snapshot DBSnapshots) ResourceName() string {
	return "rds"
}

// Snapshot names of the RDS Snapshots
func (snapshot DBSnapshots) ResourceIdentifiers() []string {
	return snapshot.SnapShots
}

// MaxBatchSize decides how many snapshots to delete in one call.
func (snapshot DBSnapshots) MaxBatchSize() int {
	return 200
}

//Nuke/Delete all snapshots
func (snapshot DBSnapshots) Nuke(session *session.Session, snapShotIdentifiers []string) error {
	if err := nukeAllRdsSnapshots(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
