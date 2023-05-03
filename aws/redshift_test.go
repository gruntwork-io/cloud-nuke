package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestNukeRedshiftClusters(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()
	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := awsSession.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	require.NoError(t, err)

	svc := redshift.New(session)

	clusterName := "test-" + strings.ToLower(util.UniqueID())

	//create cluster
	_, err = svc.CreateCluster(
		&redshift.CreateClusterInput{
			ClusterIdentifier:  aws.String(clusterName),
			MasterUsername:     aws.String("grunty"),
			MasterUserPassword: aws.String("Gruntysecurepassword1"),
			NodeType:           aws.String("dc2.large"),
			NumberOfNodes:      aws.Int64(2),
		},
	)
	require.NoError(t, err)
	err = svc.WaitUntilClusterAvailable(&redshift.DescribeClustersInput{
		ClusterIdentifier: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer svc.DeleteCluster(&redshift.DeleteClusterInput{ClusterIdentifier: aws.String(clusterName)})

	//Sleep for a minute for consistency in aws
	sleepTime, err := time.ParseDuration("1m")
	time.Sleep(sleepTime)

	//test list clusters
	clusters, err := getAllRedshiftClusters(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	//Ensure our cluster exists
	assert.Contains(t, aws.StringValueSlice(clusters), clusterName)

	//nuke cluster
	err = nukeAllRedshiftClusters(session, aws.StringSlice([]string{clusterName}))
	require.NoError(t, err)

	//check that the cluster no longer exists
	clusters, err = getAllRedshiftClusters(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(clusters), aws.StringSlice([]string{clusterName}))
}
