package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type RedshiftClustersAPI interface {
	DescribeClusters(ctx context.Context, params *redshift.DescribeClustersInput, optFns ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error)
	DeleteCluster(ctx context.Context, params *redshift.DeleteClusterInput, optFns ...func(*redshift.Options)) (*redshift.DeleteClusterOutput, error)
}
type RedshiftClusters struct {
	BaseAwsResource
	Client             RedshiftClustersAPI
	Region             string
	ClusterIdentifiers []string
}

func (rc *RedshiftClusters) Init(cfg aws.Config) {
	rc.Client = redshift.NewFromConfig(cfg)
}

func (rc *RedshiftClusters) ResourceName() string {
	return "redshift"
}

// ResourceIdentifiers - The instance names of the rds db instances
func (rc *RedshiftClusters) ResourceIdentifiers() []string {
	return rc.ClusterIdentifiers
}

func (rc *RedshiftClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (rc *RedshiftClusters) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Redshift
}

func (rc *RedshiftClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := rc.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	rc.ClusterIdentifiers = aws.ToStringSlice(identifiers)
	return rc.ClusterIdentifiers, nil
}

// Nuke - nuke 'em all!!!
func (rc *RedshiftClusters) Nuke(ctx context.Context, identifiers []string) error {
	if err := rc.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
