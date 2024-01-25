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

type DBClusters struct {
	BaseAwsResource
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (instance *DBClusters) Init(session *session.Session) {
	instance.Client = rds.New(session)
}

func (instance *DBClusters) ResourceName() string {
	return "rds-cluster"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance *DBClusters) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance *DBClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (instance *DBClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := instance.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	instance.InstanceNames = awsgo.StringValueSlice(identifiers)
	return instance.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (instance *DBClusters) Nuke(identifiers []string) error {
	if err := instance.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
