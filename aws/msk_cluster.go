package aws

import (
	"context"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllMSKClusters(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	region := session.Config.Region
	ctx := context.TODO()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(*region))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	svc := kafka.NewFromConfig(cfg)

	clusterIDs := []string{}
	paginator := kafka.NewListClustersV2Paginator(svc, &kafka.ListClustersV2Input{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range output.ClusterInfoList {
			if shouldIncludeMSKCluster(cluster, excludeAfter, configObj) {
				clusterIDs = append(clusterIDs, *cluster.ClusterArn)
			}
		}
	}

	return clusterIDs, nil
}

func shouldIncludeMSKCluster(clusterInfo types.Cluster, excludeAfter time.Time, configObj config.Config) bool {
	if clusterInfo.State == types.ClusterStateDeleting {
		return false
	}

	// if cluster is still creating, skip it as it will only throw an error when attempting to delete it
	// BadRequestException: You can't delete cluster in CREATING state.
	if clusterInfo.State == types.ClusterStateCreating {
		return false
	}

	if clusterInfo.CreationTime != nil && excludeAfter.Before(*clusterInfo.CreationTime) {
		return false
	}

	return config.ShouldInclude(
		*clusterInfo.ClusterName,
		configObj.MSKCluster.IncludeRule.NamesRegExp,
		configObj.MSKCluster.ExcludeRule.NamesRegExp,
	)
}

func nukeAllMSKClusters(session *session.Session, identifiers []string) error {
	region := session.Config.Region
	ctx := context.TODO()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(*region))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	svc := kafka.NewFromConfig(cfg)

	for _, clusterArn := range identifiers {
		_, err := svc.DeleteCluster(ctx, &kafka.DeleteClusterInput{
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
