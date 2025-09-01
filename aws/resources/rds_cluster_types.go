package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBClustersAPI interface {
	DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error)
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error)
}
type DBClusters struct {
	BaseAwsResource
	Client        DBClustersAPI
	Region        string
	InstanceNames []string
}

func (instance *DBClusters) Init(cfg aws.Config) {
	instance.Client = rds.NewFromConfig(cfg)
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

func (instance *DBClusters) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DBClusters.ResourceType
}

func (instance *DBClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := instance.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	instance.InstanceNames = aws.ToStringSlice(identifiers)
	return instance.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (instance *DBClusters) Nuke(identifiers []string) error {
	if err := instance.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
