package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (m *MSKCluster) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var clusterIDs []*string

	paginator := kafka.NewListClustersV2Paginator(m.Client, &kafka.ListClustersV2Input{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range page.ClusterInfoList {
			if m.shouldInclude(cluster, configObj) {
				clusterIDs = append(clusterIDs, cluster.ClusterArn)
			}
		}
	}

	return clusterIDs, nil
}

func (m *MSKCluster) shouldInclude(cluster types.Cluster, configObj config.Config) bool {
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

	return configObj.MSKCluster.ShouldInclude(config.ResourceValue{
		Name: cluster.ClusterName,
		Time: cluster.CreationTime,
	})
}

func (m *MSKCluster) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		return nil
	}

	for _, clusterArn := range identifiers {
		_, err := m.Client.DeleteCluster(m.Context, &kafka.DeleteClusterInput{
			ClusterArn: clusterArn,
		})
		if err != nil {
			logging.Errorf("[Failed] %s", err)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(clusterArn),
			ResourceType: "MSKCluster",
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}
