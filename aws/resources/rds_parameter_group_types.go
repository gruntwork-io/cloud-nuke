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

type RdsParameterGroup struct {
	BaseAwsResource
	Client     rdsiface.RDSAPI
	Region     string
	GroupNames []string
}

func (pg *RdsParameterGroup) Init(session *session.Session) {
	pg.Client = rds.New(session)
}

func (pg *RdsParameterGroup) ResourceName() string {
	return "rds-parameter-group"
}

// ResourceIdentifiers - The names of the rds parameter group
func (pg *RdsParameterGroup) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg *RdsParameterGroup) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (pg *RdsParameterGroup) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pg.GroupNames = awsgo.StringValueSlice(identifiers)
	return pg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (pg *RdsParameterGroup) Nuke(identifiers []string) error {
	if err := pg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
