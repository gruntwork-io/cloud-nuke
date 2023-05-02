package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
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

func createTestElasticacheReplicationGroup(t *testing.T, session *session.Session, name string) *string {
	svc := elasticache.New(session)

	params := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupDescription: awsgo.String(name),
		ReplicationGroupId:          awsgo.String(name),
		Engine:                      awsgo.String("Redis"),
		CacheNodeType:               awsgo.String("cache.r6g.large"),
	}

	validationErr := params.Validate()
	require.NoError(t, validationErr)

	output, err := svc.CreateReplicationGroup(params)
	require.NoError(t, err)

	err = svc.WaitUntilReplicationGroupAvailable(&elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: output.ReplicationGroup.ReplicationGroupId,
	})

	require.NoError(t, err)

	return output.ReplicationGroup.ReplicationGroupId
}

func TestListElasticacheClusters(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
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
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
	)

	require.NoError(t, err)

	clusterId := strings.ToLower(util.UniqueID())
	includedClusterId := "cloud-nuke-test-include-" + clusterId + "-" + strings.ToLower(util.UniqueID())
	excludedClusterId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())

	createTestElasticacheCluster(t, session, includedClusterId)
	createTestElasticacheCluster(t, session, excludedClusterId)

	// clean up after this test
	defer nukeAllElasticacheClusters(session, []*string{&includedClusterId, &excludedClusterId})

	clusterIds, err := getAllElasticacheClusters(session, region, time.Now().Add(1*time.Hour), config.Config{
		Elasticache: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile("^cloud-nuke-test-include-" + clusterId + ".*")},
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, len(clusterIds))
	assert.Contains(t, awsgo.StringValueSlice(clusterIds), includedClusterId)
}

func TestNukeElasticacheClusters(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	},
	)

	require.NoError(t, err)

	clusterId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())
	createTestElasticacheCluster(t, session, clusterId)

	replicationGroupId := "cloud-nuke-test-" + strings.ToLower(util.UniqueID())
	createTestElasticacheReplicationGroup(t, session, replicationGroupId)
	// Ensure that nukeAllElasticacheClusters can handle both scenarios for elasticache:
	// 1. The elasticache cluster is not the member of a replication group, so it can be deleted directly
	// 2. The elasticache cluster is a member of a replication group, so that replication group must be deleted
	err = nukeAllElasticacheClusters(session, []*string{&clusterId, &replicationGroupId})
	require.NoError(t, err)

	clusterIds, err := getAllElasticacheClusters(session, region, time.Now().Add(1*time.Hour), config.Config{})
	require.NoError(t, err)

	assert.NotContains(t, awsgo.StringValueSlice(clusterIds), clusterId)
}

func createParameterGroup(t *testing.T, session *session.Session) string {
	svc := elasticache.New(session)
	name := "test-" + strings.ToLower(util.UniqueID())
	input := elasticache.CreateCacheParameterGroupInput{
		CacheParameterGroupName:   awsgo.String(name),
		CacheParameterGroupFamily: awsgo.String("redis7"),
		Description:               awsgo.String("A test parameter group"),
	}
	_, err := svc.CreateCacheParameterGroup(&input)
	require.NoError(t, err)
	return name
}

func deleteParameterGroup(session *session.Session, groupName string) {
	svc := elasticache.New(session)
	svc.DeleteCacheParameterGroup(&elasticache.DeleteCacheParameterGroupInput{
		CacheParameterGroupName: awsgo.String(groupName),
	})
}

func TestNukeElasticacheParameterGroups(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()
	region, err := getRandomRegion()
	require.NoError(t, err)
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)
	//create parameter group
	paramGroup := createParameterGroup(t, session)
	defer deleteParameterGroup(session, paramGroup)
	//list parameter groups
	groups, err := getAllElasticacheParameterGroups(session, region, time.Now(), config.Config{})
	require.NoError(t, err)
	//Ensure our group exists
	assert.Contains(t, awsgo.StringValueSlice(groups), paramGroup)
	//nuke parameter groups
	err = nukeAllElasticacheParameterGroups(session, awsgo.StringSlice([]string{paramGroup}))
	require.NoError(t, err)
	//check the group no longer exists
	groups, err = getAllElasticacheParameterGroups(session, region, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(groups), paramGroup)
}

func createSubnetGroup(t *testing.T, session *session.Session) string {
	ec2Svc := ec2.New(session)
	describeVpcsParams := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("isDefault"),
				Values: []*string{awsgo.String("true")},
			},
		},
	}
	vpcs, err := ec2Svc.DescribeVpcs(describeVpcsParams)
	require.NoError(t, err)
	if len(vpcs.Vpcs) == 0 {
		err = errors.New(fmt.Sprintf("Could not find any default VPC in region %s", *session.Config.Region))
	}
	require.NoError(t, err)

	defaultVpc := vpcs.Vpcs[0]

	describeSubnetsParams := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("vpc-id"),
				Values: []*string{defaultVpc.VpcId},
			},
		},
	}
	subnets, err := ec2Svc.DescribeSubnets(describeSubnetsParams)
	require.NoError(t, err)
	if len(subnets.Subnets) == 0 {
		err = errors.New(fmt.Sprintf("Could not find any subnets for default VPC in region %s", *session.Config.Region))
	}
	require.NoError(t, err)
	var subnetIds []*string
	for _, subnet := range subnets.Subnets {
		// Only use public subnets for testing simplicity
		if !collections.ListContainsElement(AvailabilityZoneBlackList, awsgo.StringValue(subnet.AvailabilityZone)) && awsgo.BoolValue(subnet.MapPublicIpOnLaunch) {
			subnetIds = append(subnetIds, subnet.SubnetId)
		}
	}
	svc := elasticache.New(session)
	name := "test-" + strings.ToLower(util.UniqueID())
	input := elasticache.CreateCacheSubnetGroupInput{
		CacheSubnetGroupName:        awsgo.String(name),
		CacheSubnetGroupDescription: awsgo.String("A test subnet group"),
		SubnetIds:                   subnetIds,
	}
	_, err = svc.CreateCacheSubnetGroup(&input)
	require.NoError(t, err)
	return name
}

func deleteSubnetGroup(session *session.Session, groupName string) {
	svc := elasticache.New(session)
	svc.DeleteCacheSubnetGroup(&elasticache.DeleteCacheSubnetGroupInput{
		CacheSubnetGroupName: awsgo.String(groupName),
	})
}

func TestNukeElasticacheSubnetGroups(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()
	region, err := getRandomRegion()
	require.NoError(t, err)
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)
	//create subnet group
	subnetGroup := createSubnetGroup(t, session)
	defer deleteSubnetGroup(session, subnetGroup)
	//list subnet groups
	groups, err := getAllElasticacheSubnetGroups(session, region, time.Now(), config.Config{})
	require.NoError(t, err)
	//Ensure our group exists
	assert.Contains(t, awsgo.StringValueSlice(groups), subnetGroup)
	//nuke subnet groups
	err = nukeAllElasticacheSubnetGroups(session, awsgo.StringSlice([]string{subnetGroup}))
	require.NoError(t, err)
	//check the group no longer exists
	groups, err = getAllElasticacheSubnetGroups(session, region, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(groups), subnetGroup)
}
