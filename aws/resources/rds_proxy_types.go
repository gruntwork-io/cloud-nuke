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

type RdsProxy struct {
	BaseAwsResource
	Client     rdsiface.RDSAPI
	Region     string
	GroupNames []string
}

func (pg *RdsProxy) Init(session *session.Session) {
	pg.Client = rds.New(session)
}

func (pg *RdsProxy) ResourceName() string {
	return "rds-proxy"
}

// ResourceIdentifiers - The names of the rds parameter group
func (pg *RdsProxy) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg *RdsProxy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (pg *RdsProxy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.RdsProxy
}

func (pg *RdsProxy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pg.GroupNames = awsgo.StringValueSlice(identifiers)
	return pg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (pg *RdsProxy) Nuke(identifiers []string) error {
	if err := pg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
