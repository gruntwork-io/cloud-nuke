package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllRdsSnapshots(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshots []*string

	for _, database := range result.DBSnapshots {
		if database.SnapshotCreateTime != nil && excludeAfter.After(awsgo.TimeValue(database.SnapshotCreateTime)) {
			snapshots = append(snapshots, database.DBSnapshotIdentifier)
		}
	}

	return snapshots, nil
}
