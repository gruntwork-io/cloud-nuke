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

type DBSubnetGroups struct {
	Client        rdsiface.RDSAPI
	Region        string
	InstanceNames []string
}

func (dsg *DBSubnetGroups) Init(session *session.Session) {
	dsg.Client = rds.New(session)
}

func (dsg *DBSubnetGroups) ResourceName() string {
	return "rds-subnet-group"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (dsg *DBSubnetGroups) ResourceIdentifiers() []string {
	return dsg.InstanceNames
}

func (dsg *DBSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (dsg *DBSubnetGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := dsg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	dsg.InstanceNames = awsgo.StringValueSlice(identifiers)
	return dsg.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (dsg *DBSubnetGroups) Nuke(identifiers []string) error {
	if err := dsg.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
