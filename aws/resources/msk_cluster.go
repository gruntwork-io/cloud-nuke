package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/gruntwork-io/cloud-nuke/config"
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
		InitClient: WrapAwsInitClient(func(r *resource.Resource[MSKClusterAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = kafka.NewFromConfig(cfg)
		}),
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
// Skips clusters in non-deletable states: DELETING, CREATING, MAINTENANCE.
func shouldIncludeMSKCluster(cluster types.Cluster, cfg config.ResourceType) bool {
	switch cluster.State {
	case types.ClusterStateDeleting, types.ClusterStateCreating, types.ClusterStateMaintenance:
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
