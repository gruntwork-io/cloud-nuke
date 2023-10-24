package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RdsSnapshot struct {
	Client      rdsiface.RDSAPI
	Region      string
	Identifiers []string
}

func (snapshot *RdsSnapshot) Init(session *session.Session) {
	snapshot.Client = rds.New(session)
}

func (snapshot *RdsSnapshot) ResourceName() string {
	return "rds-snapshot"
}

func (snapshot *RdsSnapshot) ResourceIdentifiers() []string {
	return snapshot.Identifiers
}

func (snapshot *RdsSnapshot) MaxBatchSize() int {
	return 49
}

func (snapshot *RdsSnapshot) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := snapshot.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	snapshot.Identifiers = awsgo.StringValueSlice(identifiers)
	return snapshot.Identifiers, nil
}

// Nuke - nuke 'em all!!!
func (snapshot *RdsSnapshot) Nuke(identifiers []string) error {
	if err := snapshot.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
