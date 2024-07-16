package config

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{
		ACM:                             ResourceType{FilterRule{}, FilterRule{}, ""},
		ACMPCA:                          ResourceType{FilterRule{}, FilterRule{}, ""},
		AMI:                             ResourceType{FilterRule{}, FilterRule{}, ""},
		APIGateway:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		APIGatewayV2:                    ResourceType{FilterRule{}, FilterRule{}, ""},
		AccessAnalyzer:                  ResourceType{FilterRule{}, FilterRule{}, ""},
		AutoScalingGroup:                ResourceType{FilterRule{}, FilterRule{}, ""},
		AppRunnerService:                ResourceType{FilterRule{}, FilterRule{}, ""},
		BackupVault:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		CloudWatchAlarm:                 ResourceType{FilterRule{}, FilterRule{}, ""},
		CloudWatchDashboard:             ResourceType{FilterRule{}, FilterRule{}, ""},
		CloudWatchLogGroup:              ResourceType{FilterRule{}, FilterRule{}, ""},
		CloudtrailTrail:                 ResourceType{FilterRule{}, FilterRule{}, ""},
		CodeDeployApplications:          ResourceType{FilterRule{}, FilterRule{}, ""},
		ConfigServiceRecorder:           ResourceType{FilterRule{}, FilterRule{}, ""},
		ConfigServiceRule:               ResourceType{FilterRule{}, FilterRule{}, ""},
		DataSyncLocation:                ResourceType{FilterRule{}, FilterRule{}, ""},
		DataSyncTask:                    ResourceType{FilterRule{}, FilterRule{}, ""},
		DBGlobalClusters:                ResourceType{FilterRule{}, FilterRule{}, ""},
		DBClusters:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		DBInstances:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		DBGlobalClusterMemberships:      ResourceType{FilterRule{}, FilterRule{}, ""},
		DBSubnetGroups:                  ResourceType{FilterRule{}, FilterRule{}, ""},
		DynamoDB:                        ResourceType{FilterRule{}, FilterRule{}, ""},
		EBSVolume:                       ResourceType{FilterRule{}, FilterRule{}, ""},
		ElasticBeanstalk:                ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2:                             ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2DedicatedHosts:               ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2DHCPOption:                   ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2KeyPairs:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2IPAM:                         ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2IPAMPool:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2IPAMResourceDiscovery:        ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2IPAMScope:                    ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2Endpoint:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		EC2Subnet:                       EC2ResourceType{false, ResourceType{FilterRule{}, FilterRule{}, ""}},
		EgressOnlyInternetGateway:       ResourceType{FilterRule{}, FilterRule{}, ""},
		ECRRepository:                   ResourceType{FilterRule{}, FilterRule{}, ""},
		ECSCluster:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		ECSService:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		EKSCluster:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		ELBv1:                           ResourceType{FilterRule{}, FilterRule{}, ""},
		ELBv2:                           ResourceType{FilterRule{}, FilterRule{}, ""},
		ElasticFileSystem:               ResourceType{FilterRule{}, FilterRule{}, ""},
		ElasticIP:                       ResourceType{FilterRule{}, FilterRule{}, ""},
		Elasticache:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		ElasticacheParameterGroups:      ResourceType{FilterRule{}, FilterRule{}, ""},
		ElasticacheSubnetGroups:         ResourceType{FilterRule{}, FilterRule{}, ""},
		GuardDuty:                       ResourceType{FilterRule{}, FilterRule{}, ""},
		IAMGroups:                       ResourceType{FilterRule{}, FilterRule{}, ""},
		IAMPolicies:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		IAMRoles:                        ResourceType{FilterRule{}, FilterRule{}, ""},
		IAMServiceLinkedRoles:           ResourceType{FilterRule{}, FilterRule{}, ""},
		IAMUsers:                        ResourceType{FilterRule{}, FilterRule{}, ""},
		KMSCustomerKeys:                 KMSCustomerKeyResourceType{false, ResourceType{}},
		KinesisStream:                   ResourceType{FilterRule{}, FilterRule{}, ""},
		KinesisFirehose:                 ResourceType{FilterRule{}, FilterRule{}, ""},
		LambdaFunction:                  ResourceType{FilterRule{}, FilterRule{}, ""},
		LambdaLayer:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		LaunchConfiguration:             ResourceType{FilterRule{}, FilterRule{}, ""},
		LaunchTemplate:                  ResourceType{FilterRule{}, FilterRule{}, ""},
		MacieMember:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		MSKCluster:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		NatGateway:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		OIDCProvider:                    ResourceType{FilterRule{}, FilterRule{}, ""},
		OpenSearchDomain:                ResourceType{FilterRule{}, FilterRule{}, ""},
		Redshift:                        ResourceType{FilterRule{}, FilterRule{}, ""},
		RdsSnapshot:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		RdsParameterGroup:               ResourceType{FilterRule{}, FilterRule{}, ""},
		RdsProxy:                        ResourceType{FilterRule{}, FilterRule{}, ""},
		S3:                              ResourceType{FilterRule{}, FilterRule{}, ""},
		S3AccessPoint:                   ResourceType{FilterRule{}, FilterRule{}, ""},
		S3ObjectLambdaAccessPoint:       ResourceType{FilterRule{}, FilterRule{}, ""},
		S3MultiRegionAccessPoint:        ResourceType{FilterRule{}, FilterRule{}, ""},
		SESIdentity:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		SESConfigurationSet:             ResourceType{FilterRule{}, FilterRule{}, ""},
		SESReceiptRuleSet:               ResourceType{FilterRule{}, FilterRule{}, ""},
		SESReceiptFilter:                ResourceType{FilterRule{}, FilterRule{}, ""},
		SESEmailTemplates:               ResourceType{FilterRule{}, FilterRule{}, ""},
		SNS:                             ResourceType{FilterRule{}, FilterRule{}, ""},
		SQS:                             ResourceType{FilterRule{}, FilterRule{}, ""},
		SageMakerNotebook:               ResourceType{FilterRule{}, FilterRule{}, ""},
		SecretsManagerSecrets:           ResourceType{FilterRule{}, FilterRule{}, ""},
		SecurityHub:                     ResourceType{FilterRule{}, FilterRule{}, ""},
		Snapshots:                       ResourceType{FilterRule{}, FilterRule{}, ""},
		TransitGateway:                  ResourceType{FilterRule{}, FilterRule{}, ""},
		TransitGatewayRouteTable:        ResourceType{FilterRule{}, FilterRule{}, ""},
		TransitGatewaysVpcAttachment:    ResourceType{FilterRule{}, FilterRule{}, ""},
		TransitGatewayPeeringAttachment: ResourceType{FilterRule{}, FilterRule{}, ""},
		VPC:                             EC2ResourceType{false, ResourceType{FilterRule{}, FilterRule{}, ""}},
		Route53HostedZone:               ResourceType{FilterRule{}, FilterRule{}, ""},
		Route53CIDRCollection:           ResourceType{FilterRule{}, FilterRule{}, ""},
		Route53TrafficPolicy:            ResourceType{FilterRule{}, FilterRule{}, ""},
		InternetGateway:                 ResourceType{FilterRule{}, FilterRule{}, ""},
		NetworkACL:                      ResourceType{FilterRule{}, FilterRule{}, ""},
		NetworkInterface:                ResourceType{FilterRule{}, FilterRule{}, ""},
		SecurityGroup:                   EC2ResourceType{false, ResourceType{FilterRule{}, FilterRule{}, ""}},
		NetworkFirewall:                 ResourceType{FilterRule{}, FilterRule{}, ""},
		NetworkFirewallPolicy:           ResourceType{FilterRule{}, FilterRule{}, ""},
		NetworkFirewallRuleGroup:        ResourceType{FilterRule{}, FilterRule{}, ""},
		NetworkFirewallTLSConfig:        ResourceType{FilterRule{}, FilterRule{}, ""},
		NetworkFirewallResourcePolicy:   ResourceType{FilterRule{}, FilterRule{}, ""},
		VPCLatticeServiceNetwork:        ResourceType{FilterRule{}, FilterRule{}, ""},
		VPCLatticeService:               ResourceType{FilterRule{}, FilterRule{}, ""},
		VPCLatticeTargetGroup:           ResourceType{FilterRule{}, FilterRule{}, ""},
	}
}

func TestConfig_Garbage(t *testing.T) {
	configFilePath := "./mocks/garbage.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfig_Malformed(t *testing.T) {
	configFilePath := "./mocks/malformed.yaml"
	_, err := GetConfig(configFilePath)

	// Expect malformed to throw a yaml TypeError
	require.Error(t, err, "Received expected error")
	return
}

func TestConfig_Empty(t *testing.T) {
	configFilePath := "./mocks/empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestShouldInclude_AllowWhenEmpty(t *testing.T) {
	var includeREs []Expression
	var excludeREs []Expression

	assert.True(t, ShouldInclude("test-open-vpn", includeREs, excludeREs),
		"Should include when both lists are empty")
}

func TestShouldInclude_ExcludeWhenMatches(t *testing.T) {
	var includeREs []Expression

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.False(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should not include when matches from the 'exclude' list")
	assert.True(t, ShouldInclude("tf-state-bucket", includeREs, excludeREs),
		"Should include when doesn't matches from the 'exclude' list")
}

func TestShouldInclude_IncludeWhenMatches(t *testing.T) {
	include, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	var excludeREs []Expression

	assert.True(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should include when matches the 'include' list")
	assert.False(t, ShouldInclude("test-vpc-123", includeREs, excludeREs),
		"Should not include when doesn't matches the 'include' list")
}

func TestShouldInclude_WhenMatchesIncludeAndExclude(t *testing.T) {
	include, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	exclude, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.True(t, ShouldInclude("test-eks-cluster-123", includeREs, excludeREs),
		"Should include when matches the 'include' list but not matches the 'exclude' list")
	assert.False(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should not include when matches 'exclude' list")
	assert.False(t, ShouldInclude("terraform-tf-state", includeREs, excludeREs),
		"Should not include when doesn't matches 'include' list")
}

func TestShouldIncludeBasedOnTime_IncludeTimeBefore(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		IncludeRule: FilterRule{TimeBefore: &now},
	}
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_IncludeTimeAfter(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		IncludeRule: FilterRule{TimeAfter: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_ExcludeTimeBefore(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		ExcludeRule: FilterRule{TimeBefore: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_ExcludeTimeAfter(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		ExcludeRule: FilterRule{TimeAfter: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
}

func TestShouldInclude_NameAndTimeFilter(t *testing.T) {
	now := time.Now()

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}
	r := ResourceType{
		ExcludeRule: FilterRule{
			NamesRegExp: excludeREs,
			TimeAfter:   &now,
		},
	}

	// Filter by Time
	assert.False(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("hello_world"),
		Time: aws.Time(now.Add(1)),
	}))
	// Filter by Name
	assert.False(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("test_hello_world"),
		Time: aws.Time(now.Add(1)),
	}))
	// Pass filters
	assert.True(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("hello_world"),
		Time: aws.Time(now.Add(-1)),
	}))
}

func TestAddIncludeAndExcludeAfterTime(t *testing.T) {
	now := aws.Time(time.Now())

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	testConfig := &Config{}
	testConfig.ACM = ResourceType{
		ExcludeRule: FilterRule{
			NamesRegExp: excludeREs,
			TimeAfter:   now,
		},
	}

	testConfig.AddExcludeAfterTime(now)
	assert.Equal(t, testConfig.ACM.ExcludeRule.NamesRegExp, excludeREs)
	assert.Equal(t, testConfig.ACM.ExcludeRule.TimeAfter, now)
	assert.Nil(t, testConfig.ACM.IncludeRule.TimeAfter)

	testConfig.AddIncludeAfterTime(now)
	assert.Equal(t, testConfig.ACM.ExcludeRule.NamesRegExp, excludeREs)
	assert.Equal(t, testConfig.ACM.ExcludeRule.TimeAfter, now)
	assert.Equal(t, testConfig.ACM.IncludeRule.TimeAfter, now)
}
