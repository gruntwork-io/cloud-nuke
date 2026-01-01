package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// MSKClusterAPI defines the interface for MSK Cluster operations.
type MSKClusterAPI interface {
	ListClustersV2(ctx context.Context, params *kafka.ListClustersV2Input, optFns ...func(*kafka.Options)) (*kafka.ListClustersV2Output, error)
	DeleteCluster(ctx context.Context, params *kafka.DeleteClusterInput, optFns ...func(*kafka.Options)) (*kafka.DeleteClusterOutput, error)
}

// NewMSKCluster creates a new MSKCluster resource using the generic resource pattern.
func NewMSKCluster() AwsResource {
	return NewAwsResource(&resource.Resource[MSKClusterAPI]{
		ResourceTypeName: "msk-cluster",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[MSKClusterAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for MSKCluster client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = kafka.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.MSKCluster
		},
		Lister: listMSKClusters,
		Nuker:  resource.SimpleBatchDeleter(deleteMSKCluster),
	})
}

// listMSKClusters retrieves all MSK clusters that match the config filters.
func listMSKClusters(ctx context.Context, client MSKClusterAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var clusterArns []*string

	paginator := kafka.NewListClustersV2Paginator(client, &kafka.ListClustersV2Input{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range page.ClusterInfoList {
			if shouldIncludeMSKCluster(cluster, cfg) {
				clusterArns = append(clusterArns, cluster.ClusterArn)
			}
		}
	}

	return clusterArns, nil
}

// shouldIncludeMSKCluster determines if a cluster should be included based on state and config.
func shouldIncludeMSKCluster(cluster types.Cluster, cfg config.ResourceType) bool {
	if cluster.State == types.ClusterStateDeleting {
		return false
	}

	// if cluster is still creating, skip it as it will only throw an error when attempting to delete it
	// BadRequestException: You can't delete cluster in CREATING state.
	if cluster.State == types.ClusterStateCreating {
		return false
	}

	// if cluster is in maintenance, skip it as it will only throw an error when attempting to delete it
	// BadRequestException: You can't delete cluster in MAINTENANCE state.
	if cluster.State == types.ClusterStateMaintenance {
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: cluster.ClusterName,
		Time: cluster.CreationTime,
	})
}

// deleteMSKCluster deletes a single MSK cluster.
func deleteMSKCluster(ctx context.Context, client MSKClusterAPI, clusterArn *string) error {
	_, err := client.DeleteCluster(ctx, &kafka.DeleteClusterInput{
		ClusterArn: clusterArn,
	})
	return err
}
