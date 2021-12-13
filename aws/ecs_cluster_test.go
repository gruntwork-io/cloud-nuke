package aws

import (
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
)

// Test we can create a cluster, tag it, and then find the tag
func TestCanTagEcsClusters(t *testing.T) {
	t.Parallel()
	region := getRandomFargateSupportedRegion()

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	cluster := createEcsFargateCluster(t, awsSession, "cloud-nuke-test-"+util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster)

	tagValue := time.Now().UTC()

	tagErr := tagEcsClusterWhenFirstSeen(awsSession, cluster.ClusterArn, tagValue)
	require.NoError(t, tagErr)

	returnedTag, err := getFirstSeenEcsClusterTag(awsSession, cluster.ClusterArn)
	require.NoError(t, err)

	parsedTagValue, parseErr1 := parseTimestampTag(formatTimestampTag(tagValue))
	require.NoError(t, parseErr1)

	parsedReturnValue, parseErr2 := parseTimestampTag(formatTimestampTag(returnedTag))
	require.NoError(t, parseErr2)

	//compare that the tags' Time values after formatting are equal
	assert.Equal(t, parsedTagValue, parsedReturnValue)
}

// Test we can get all ECS clusters younger than < X time based on tags
func TestCanListAllEcsClustersOlderThan24hours(t *testing.T) {
	t.Parallel()
	region := getRandomFargateSupportedRegion()

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	cluster1 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster1)
	cluster2 := createEcsFargateCluster(t, awsSession, util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster2)

	now := time.Now().UTC()
	var olderClusterTagValue = now.Add(time.Hour * time.Duration(-48))
	var youngerClusterTagValue = now.Add(time.Hour * time.Duration(-23))

	err1 := tagEcsClusterWhenFirstSeen(awsSession, cluster1.ClusterArn, olderClusterTagValue)
	require.NoError(t, err1)
	err2 := tagEcsClusterWhenFirstSeen(awsSession, cluster2.ClusterArn, youngerClusterTagValue)
	require.NoError(t, err2)

	last24Hours := now.Add(time.Hour * time.Duration(-24))
	filteredClusterArns, err := getAllEcsClustersOlderThan(awsSession, last24Hours, config.Config{})
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(filteredClusterArns), awsgo.StringValue(cluster1.ClusterArn))
}

// Test we can nuke all ECS clusters older than 24hrs
func TestCanNukeAllEcsClustersOlderThan24Hours(t *testing.T) {
	t.Parallel()
	region := getRandomFargateSupportedRegion()

	awsSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	cluster1 := createEcsFargateCluster(t, awsSession, "cloud-nuke-test-"+util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster1)
	cluster2 := createEcsFargateCluster(t, awsSession, "cloud-nuke-test-"+util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster2)
	cluster3 := createEcsFargateCluster(t, awsSession, "cloud-nuke-test-"+util.UniqueID())
	defer deleteEcsCluster(awsSession, cluster3)

	now := time.Now().UTC()
	var oldClusterTagValue1 = now.Add(time.Hour * time.Duration(-48))
	var youngClusterTagValue = now
	var oldClusterTagValue2 = now.Add(time.Hour * time.Duration(-27))

	err1 := tagEcsClusterWhenFirstSeen(awsSession, cluster1.ClusterArn, oldClusterTagValue1)
	require.NoError(t, err1)
	err2 := tagEcsClusterWhenFirstSeen(awsSession, cluster2.ClusterArn, youngClusterTagValue)
	require.NoError(t, err2)
	err3 := tagEcsClusterWhenFirstSeen(awsSession, cluster3.ClusterArn, oldClusterTagValue2)
	require.NoError(t, err3)

	last24Hours := now.Add(time.Hour * time.Duration(-24))
	filteredClusterArns, err := getAllEcsClustersOlderThan(awsSession, last24Hours, config.Config{})
	require.NoError(t, err)

	nukeErr := nukeEcsClusters(awsSession, filteredClusterArns)
	require.NoError(t, nukeErr)

	allLeftClusterArns, err := getAllEcsClusters(awsSession)
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(allLeftClusterArns), awsgo.StringValue(cluster2.ClusterArn))
}

// Test the config file filtering works as expected
func TestShouldIncludeECSCluster(t *testing.T) {
	mockCluster := &ecs.Cluster{
		ClusterName: awsgo.String("cloud-nuke-test"),
		Status:      awsgo.String("ACTIVE"),
	}

	mockClusterInactive := &ecs.Cluster{
		ClusterName: awsgo.String("cloud-nuke-test"),
		Status:      awsgo.String("INACTIVE"),
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		logging.Logger.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		ECSCluster: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	mockIncludeConfig := config.Config{
		ECSCluster: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	cases := []struct {
		Name     string
		Cluster  *ecs.Cluster
		Config   config.Config
		Expected bool
	}{
		{
			Name:     "ConfigExclude",
			Cluster:  mockCluster,
			Config:   mockExcludeConfig,
			Expected: false,
		},
		{
			Name:     "ConfigInclude",
			Cluster:  mockCluster,
			Config:   mockIncludeConfig,
			Expected: true,
		},
		{
			Name:     "ConfigIncludeInactive",
			Cluster:  mockClusterInactive,
			Config:   mockIncludeConfig,
			Expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeECSCluster(c.Cluster, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}
