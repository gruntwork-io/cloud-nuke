package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBGlobalClustersAPI interface {
	DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error)
	DeleteGlobalCluster(ctx context.Context, params *rds.DeleteGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteGlobalClusterOutput, error)
}

type DBGlobalClusters struct {
	BaseAwsResource
	Client        DBGlobalClustersAPI
	Region        string
	InstanceNames []string
}

func (instance *DBGlobalClusters) Init(cfg aws.Config) {
	instance.Client = rds.NewFromConfig(cfg)
}

func (instance *DBGlobalClusters) ResourceName() string {
	return "rds-global-cluster"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance *DBGlobalClusters) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance *DBGlobalClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (instance *DBGlobalClusters) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DBGlobalClusters
}

func (instance *DBGlobalClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := instance.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	instance.InstanceNames = aws.ToStringSlice(identifiers)
	return instance.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (instance *DBGlobalClusters) Nuke(identifiers []string) error {
	if err := instance.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
