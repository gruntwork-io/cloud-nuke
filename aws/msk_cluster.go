package aws

import (
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func (m MSKCluster) getAll(configObj config.Config) ([]string, error) {
	clusterIDs := []string{}

	err := m.Client.ListClustersV2Pages(&kafka.ListClustersV2Input{}, func(page *kafka.ListClustersV2Output, lastPage bool) bool {
		for _, cluster := range page.ClusterInfoList {
			if m.shouldInclude(cluster, configObj) {
				clusterIDs = append(clusterIDs, *cluster.ClusterArn)
			}
		}
		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	return clusterIDs, nil
}

func (m MSKCluster) shouldInclude(cluster *kafka.Cluster, configObj config.Config) bool {
	if *cluster.State == kafka.ClusterStateDeleting {
		return false
	}

	// if cluster is still creating, skip it as it will only throw an error when attempting to delete it
	// BadRequestException: You can't delete cluster in CREATING state.
	if *cluster.State == kafka.ClusterStateCreating {
		return false
	}

	return configObj.MSKCluster.ShouldInclude(config.ResourceValue{
		Name: cluster.ClusterName,
		Time: cluster.CreationTime,
	})
}

func (m MSKCluster) nukeAll(identifiers []string) error {
	for _, clusterArn := range identifiers {
		_, err := m.Client.DeleteCluster(&kafka.DeleteClusterInput{
			ClusterArn: &clusterArn,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   clusterArn,
			ResourceType: "MSKCluster",
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}
