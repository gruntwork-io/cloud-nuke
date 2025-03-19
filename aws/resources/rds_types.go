package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RDSAPI interface {
	ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error)
	DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
}
type DBInstances struct {
	BaseAwsResource
	Client        RDSAPI
	Region        string
	InstanceNames []string
}

func (di *DBInstances) Init(cfg aws.Config) {
	di.Client = rds.NewFromConfig(cfg)
}

func (di *DBInstances) ResourceName() string {
	return "rds"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (di *DBInstances) ResourceIdentifiers() []string {
	return di.InstanceNames
}

func (di *DBInstances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (di *DBInstances) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DBInstances.ResourceType
}

func (di *DBInstances) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := di.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	di.InstanceNames = aws.ToStringSlice(identifiers)
	return di.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (di *DBInstances) Nuke(identifiers []string) error {
	if err := di.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type RdsDeleteError struct {
	name string
}

func (e RdsDeleteError) Error() string {
	return "RDS DB Instance:" + e.name + "was not deleted"
}
