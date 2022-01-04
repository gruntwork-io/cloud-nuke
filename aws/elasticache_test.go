package aws

import (
	"strings"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
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
	if err != nil {
		assert.Failf(t, "Could not create test Elasticache cluster", errors.WithStackTrace(err).Error())
	}
}

func TestListElasticacheClusters(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	clusterId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())
	createTestElasticacheCluster(t, session, clusterId)

	// clean up after this test
	defer nukeAllElasticacheClusters(session, []*string{&clusterId})

	clusterIds, err := getAllElasticacheClusters(session, region)
	if err != nil {
		assert.Failf(t, "Unable to fetch list of Elasticache clusters", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(clusterIds), clusterId)
}
