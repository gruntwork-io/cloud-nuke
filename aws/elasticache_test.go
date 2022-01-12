package aws

import (
	"regexp"
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestElasticacheCluster(t *testing.T, session *session.Session, name string) {
	svc := elasticache.New(session)

	param := elasticache.CreateCacheClusterInput{
		CacheClusterId: awsgo.String(name),
		CacheNodeType:  awsgo.String("cache.t3.micro"),
		Engine:         awsgo.String("memcached"),
		NumCacheNodes:  awsgo.Int64(1),
	}

	_, err := svc.CreateCacheCluster(&param)
	require.NoError(t, err)

	err = svc.WaitUntilCacheClusterAvailable(&elasticache.DescribeCacheClustersInput{
		CacheClusterId: awsgo.String(name),
	})

	require.NoError(t, err)
}

func TestListElasticacheClusters(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	clusterId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())
	createTestElasticacheCluster(t, session, clusterId)

	// clean up after this test
	defer nukeAllElasticacheClusters(session, []*string{&clusterId})

	clusterIds, err := getAllElasticacheClusters(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	assert.Contains(t, awsgo.StringValueSlice(clusterIds), clusterId)
}

func TestListElasticacheClustersWithConfigFile(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	includedClusterId := "cloud-nuke-test-include-" + strings.ToLower(util.UniqueID())
	excludedClusterId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())

	createTestElasticacheCluster(t, session, includedClusterId)
	createTestElasticacheCluster(t, session, excludedClusterId)

	// clean up after this test
	defer nukeAllElasticacheClusters(session, []*string{&includedClusterId, &excludedClusterId})

	clusterIds, err := getAllElasticacheClusters(session, region, time.Now().Add(1*time.Hour), config.Config{
		Elasticache: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile("^cloud-nuke-test-include-.*")},
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, len(clusterIds))
	assert.Contains(t, awsgo.StringValueSlice(clusterIds), includedClusterId)
}

func TestNukeElasticacheClusters(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	require.NoError(t, err)

	clusterId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())
	createTestElasticacheCluster(t, session, clusterId)

	err = nukeAllElasticacheClusters(session, []*string{&clusterId})
	require.NoError(t, err)

	clusterIds, err := getAllElasticacheClusters(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(clusterIds), clusterId)
}
