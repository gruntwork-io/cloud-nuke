package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DBGCMembershipsAPI interface {
	DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error)
	RemoveFromGlobalCluster(ctx context.Context, params *rds.RemoveFromGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.RemoveFromGlobalClusterOutput, error)
}

type DBGlobalClusterMemberships struct {
	BaseAwsResource
	Client        DBGCMembershipsAPI
	Region        string
	InstanceNames []string
}

func (instance *DBGlobalClusterMemberships) Init(cfg aws.Config) {
	instance.Client = rds.NewFromConfig(cfg)
}

func (instance *DBGlobalClusterMemberships) ResourceName() string {
	return "rds-global-cluster-membership"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (instance *DBGlobalClusterMemberships) ResourceIdentifiers() []string {
	return instance.InstanceNames
}

func (instance *DBGlobalClusterMemberships) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (instance *DBGlobalClusterMemberships) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DBGlobalClusterMemberships
}

func (instance *DBGlobalClusterMemberships) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := instance.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	instance.InstanceNames = aws.ToStringSlice(identifiers)
	return instance.InstanceNames, nil
}

// Nuke - nuke 'em all!!!
func (instance *DBGlobalClusterMemberships) Nuke(ctx context.Context, identifiers []string) error {
	if err := instance.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
