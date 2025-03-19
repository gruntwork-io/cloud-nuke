package config

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{
		ACM:                             ResourceType{FilterRule{}, FilterRule{}, "", false},
		ACMPCA:                          ResourceType{FilterRule{}, FilterRule{}, "", false},
		AMI:                             ResourceType{FilterRule{}, FilterRule{}, "", false},
		APIGateway:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		APIGatewayV2:                    ResourceType{FilterRule{}, FilterRule{}, "", false},
		AccessAnalyzer:                  ResourceType{FilterRule{}, FilterRule{}, "", false},
		AutoScalingGroup:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		AppRunnerService:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		BackupVault:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		ManagedPrometheus:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		CloudWatchAlarm:                 ResourceType{FilterRule{}, FilterRule{}, "", false},
		CloudWatchDashboard:             ResourceType{FilterRule{}, FilterRule{}, "", false},
		CloudWatchLogGroup:              ResourceType{FilterRule{}, FilterRule{}, "", false},
		CloudtrailTrail:                 ResourceType{FilterRule{}, FilterRule{}, "", false},
		CodeDeployApplications:          ResourceType{FilterRule{}, FilterRule{}, "", false},
		ConfigServiceRecorder:           ResourceType{FilterRule{}, FilterRule{}, "", false},
		ConfigServiceRule:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		DataSyncLocation:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		DataSyncTask:                    ResourceType{FilterRule{}, FilterRule{}, "", false},
		DBGlobalClusters:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		DBClusters:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		DBInstances:                     AWSProtectectableResourceType{ResourceType: ResourceType{FilterRule{}, FilterRule{}, "", false}},
		DBGlobalClusterMemberships:      ResourceType{FilterRule{}, FilterRule{}, "", false},
		DBSubnetGroups:                  ResourceType{FilterRule{}, FilterRule{}, "", false},
		DynamoDB:                        ResourceType{FilterRule{}, FilterRule{}, "", false},
		EBSVolume:                       ResourceType{FilterRule{}, FilterRule{}, "", false},
		ElasticBeanstalk:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2:                             ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2DedicatedHosts:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2DHCPOption:                   ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2KeyPairs:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2IPAM:                         ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2IPAMPool:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2IPAMResourceDiscovery:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2IPAMScope:                    ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2Endpoint:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		EC2Subnet:                       EC2ResourceType{false, ResourceType{FilterRule{}, FilterRule{}, "", false}},
		EC2PlacementGroups:              ResourceType{FilterRule{}, FilterRule{}, "", false},
		EgressOnlyInternetGateway:       ResourceType{FilterRule{}, FilterRule{}, "", false},
		ECRRepository:                   ResourceType{FilterRule{}, FilterRule{}, "", false},
		ECSCluster:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		ECSService:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		EKSCluster:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		ELBv1:                           ResourceType{FilterRule{}, FilterRule{}, "", false},
		ELBv2:                           ResourceType{FilterRule{}, FilterRule{}, "", false},
		ElasticFileSystem:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		ElasticIP:                       ResourceType{FilterRule{}, FilterRule{}, "", false},
		Elasticache:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		ElasticacheParameterGroups:      ResourceType{FilterRule{}, FilterRule{}, "", false},
		ElasticCacheServerless:          ResourceType{FilterRule{}, FilterRule{}, "", false},
		ElasticacheSubnetGroups:         ResourceType{FilterRule{}, FilterRule{}, "", false},
		EventBridge:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		EventBridgeArchive:              ResourceType{FilterRule{}, FilterRule{}, "", false},
		EventBridgeRule:                 ResourceType{FilterRule{}, FilterRule{}, "", false},
		EventBridgeSchedule:             ResourceType{FilterRule{}, FilterRule{}, "", false},
		EventBridgeScheduleGroup:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		Grafana:                         ResourceType{FilterRule{}, FilterRule{}, "", false},
		GuardDuty:                       ResourceType{FilterRule{}, FilterRule{}, "", false},
		IAMGroups:                       ResourceType{FilterRule{}, FilterRule{}, "", false},
		IAMPolicies:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		IAMRoles:                        ResourceType{FilterRule{}, FilterRule{}, "", false},
		IAMServiceLinkedRoles:           ResourceType{FilterRule{}, FilterRule{}, "", false},
		IAMInstanceProfiles:             ResourceType{FilterRule{}, FilterRule{}, "", false},
		IAMUsers:                        ResourceType{FilterRule{}, FilterRule{}, "", false},
		KMSCustomerKeys:                 KMSCustomerKeyResourceType{false, ResourceType{}},
		KinesisStream:                   ResourceType{FilterRule{}, FilterRule{}, "", false},
		KinesisFirehose:                 ResourceType{FilterRule{}, FilterRule{}, "", false},
		LambdaFunction:                  ResourceType{FilterRule{}, FilterRule{}, "", false},
		LambdaLayer:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		LaunchConfiguration:             ResourceType{FilterRule{}, FilterRule{}, "", false},
		LaunchTemplate:                  ResourceType{FilterRule{}, FilterRule{}, "", false},
		MacieMember:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		MSKCluster:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		NatGateway:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		OIDCProvider:                    ResourceType{FilterRule{}, FilterRule{}, "", false},
		OpenSearchDomain:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		Redshift:                        ResourceType{FilterRule{}, FilterRule{}, "", false},
		RdsSnapshot:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		RdsParameterGroup:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		RdsProxy:                        ResourceType{FilterRule{}, FilterRule{}, "", false},
		S3:                              ResourceType{FilterRule{}, FilterRule{}, "", false},
		S3AccessPoint:                   ResourceType{FilterRule{}, FilterRule{}, "", false},
		S3ObjectLambdaAccessPoint:       ResourceType{FilterRule{}, FilterRule{}, "", false},
		S3MultiRegionAccessPoint:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		SESIdentity:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		SESConfigurationSet:             ResourceType{FilterRule{}, FilterRule{}, "", false},
		SESReceiptRuleSet:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		SESReceiptFilter:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		SESEmailTemplates:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		SNS:                             ResourceType{FilterRule{}, FilterRule{}, "", false},
		SQS:                             ResourceType{FilterRule{}, FilterRule{}, "", false},
		SageMakerNotebook:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		SageMakerStudioDomain:           ResourceType{FilterRule{}, FilterRule{}, "", false},
		SecretsManagerSecrets:           ResourceType{FilterRule{}, FilterRule{}, "", false},
		SecurityHub:                     ResourceType{FilterRule{}, FilterRule{}, "", false},
		Snapshots:                       ResourceType{FilterRule{}, FilterRule{}, "", false},
		TransitGateway:                  ResourceType{FilterRule{}, FilterRule{}, "", false},
		TransitGatewayRouteTable:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		TransitGatewaysVpcAttachment:    ResourceType{FilterRule{}, FilterRule{}, "", false},
		TransitGatewayPeeringAttachment: ResourceType{FilterRule{}, FilterRule{}, "", false},
		VPC:                             EC2ResourceType{false, ResourceType{FilterRule{}, FilterRule{}, "", false}},
		Route53HostedZone:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		Route53CIDRCollection:           ResourceType{FilterRule{}, FilterRule{}, "", false},
		Route53TrafficPolicy:            ResourceType{FilterRule{}, FilterRule{}, "", false},
		InternetGateway:                 ResourceType{FilterRule{}, FilterRule{}, "", false},
		NetworkACL:                      ResourceType{FilterRule{}, FilterRule{}, "", false},
		NetworkInterface:                ResourceType{FilterRule{}, FilterRule{}, "", false},
		SecurityGroup:                   EC2ResourceType{false, ResourceType{FilterRule{}, FilterRule{}, "", false}},
		NetworkFirewall:                 ResourceType{FilterRule{}, FilterRule{}, "", false},
		NetworkFirewallPolicy:           ResourceType{FilterRule{}, FilterRule{}, "", false},
		NetworkFirewallRuleGroup:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		NetworkFirewallTLSConfig:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		NetworkFirewallResourcePolicy:   ResourceType{FilterRule{}, FilterRule{}, "", false},
		VPCLatticeServiceNetwork:        ResourceType{FilterRule{}, FilterRule{}, "", false},
		VPCLatticeService:               ResourceType{FilterRule{}, FilterRule{}, "", false},
		VPCLatticeTargetGroup:           ResourceType{FilterRule{}, FilterRule{}, "", false},
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

	assert.True(t, ShouldInclude(aws.String("test-open-vpn"), includeREs, excludeREs),
		"Should include when both lists are empty")
}

func TestShouldInclude_ExcludeWhenMatches(t *testing.T) {
	var includeREs []Expression

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.False(t, ShouldInclude(aws.String("test-openvpn-123"), includeREs, excludeREs),
		"Should not include when matches from the 'exclude' list")
	assert.True(t, ShouldInclude(aws.String("tf-state-bucket"), includeREs, excludeREs),
		"Should include when doesn't matches from the 'exclude' list")
}

func TestShouldInclude_IncludeWhenMatches(t *testing.T) {
	include, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	var excludeREs []Expression

	assert.True(t, ShouldInclude(aws.String("test-openvpn-123"), includeREs, excludeREs),
		"Should include when matches the 'include' list")
	assert.False(t, ShouldInclude(aws.String("test-vpc-123"), includeREs, excludeREs),
		"Should not include when doesn't matches the 'include' list")
}

func TestShouldInclude_WhenMatchesIncludeAndExclude(t *testing.T) {
	include, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	exclude, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.True(t, ShouldInclude(aws.String("test-eks-cluster-123"), includeREs, excludeREs),
		"Should include when matches the 'include' list but not matches the 'exclude' list")
	assert.False(t, ShouldInclude(aws.String("test-openvpn-123"), includeREs, excludeREs),
		"Should not include when matches 'exclude' list")
	assert.False(t, ShouldInclude(aws.String("terraform-tf-state"), includeREs, excludeREs),
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

func TestGetExclusionTag(t *testing.T) {
	tests := []struct {
		name        string
		want        string
		ExcludeRule FilterRule
	}{
		{
			name: "empty config returns default tag",
			want: DefaultAwsResourceExclusionTagKey,
		},
		{
			name: "custom exclusion tag is returned",
			ExcludeRule: FilterRule{
				Tag: aws.String("my-custom-tag"),
			},
			want: "my-custom-tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := &Config{}
			testConfig.ACM = ResourceType{
				ExcludeRule: tt.ExcludeRule,
			}

			require.Equal(t, tt.want, testConfig.ACM.getExclusionTag())
		})
	}
}

func TestShouldIncludeBasedOnTag(t *testing.T) {
	timeIn2h := time.Now().Add(2 * time.Hour)

	type arg struct {
		ExcludeRule        FilterRule
		ProtectUntilExpire bool
	}
	tests := []struct {
		name   string
		given  arg
		when   map[string]string
		expect bool
	}{
		{
			name:   "should include resource with default exclude tag",
			given:  arg{},
			when:   map[string]string{DefaultAwsResourceExclusionTagKey: "true"},
			expect: false,
		},
		{
			name: "should include resource with custom exclude tag",
			given: arg{
				ExcludeRule: FilterRule{
					Tag: aws.String("my-custom-skip-tag"),
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": "true"},
			expect: false,
		},
		{
			name: "should include resource with custom exclude tag and empty value",
			given: arg{
				ExcludeRule: FilterRule{
					Tag: aws.String("my-custom-skip-tag"),
					TagValue: &Expression{
						RE: *regexp.MustCompile(""),
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": ""},
			expect: false,
		},
		{
			name: "should include resource with custom exclude tag and empty value (using regular expression)",
			given: arg{
				ExcludeRule: FilterRule{
					Tag: aws.String("my-custom-skip-tag"),
					TagValue: &Expression{
						RE: *regexp.MustCompile(".*"),
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": ""},
			expect: false,
		},
		{
			name: "should include resource with custom exclude tag and prefix value (using regular expression)",
			given: arg{
				ExcludeRule: FilterRule{
					Tag: aws.String("my-custom-skip-tag"),
					TagValue: &Expression{
						RE: *regexp.MustCompile("protected-.*"),
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": "protected-database"},
			expect: false,
		},
		{
			name: "should include resource when exclude tag is not set to true",
			given: arg{
				ExcludeRule: FilterRule{
					Tag: aws.String(DefaultAwsResourceExclusionTagKey),
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{DefaultAwsResourceExclusionTagKey: "false"},
			expect: true,
		},
		{
			name: "should include resource when no tags are set",
			given: arg{
				ExcludeRule: FilterRule{
					Tag: aws.String(DefaultAwsResourceExclusionTagKey),
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{},
			expect: true,
		},
		{
			name: "should include resource when protection expires in the future",
			given: arg{
				ExcludeRule:        FilterRule{},
				ProtectUntilExpire: true,
			},
			when:   map[string]string{CloudNukeAfterExclusionTagKey: timeIn2h.Format(time.RFC3339)},
			expect: false,
		},
		{
			name: "should include resource when protection expired in the past",
			given: arg{
				ExcludeRule:        FilterRule{},
				ProtectUntilExpire: true,
			},
			when:   map[string]string{CloudNukeAfterExclusionTagKey: time.Now().Format(time.RFC3339)},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ResourceType{
				ExcludeRule:        tt.given.ExcludeRule,
				ProtectUntilExpire: tt.given.ProtectUntilExpire,
			}

			require.Equal(t, tt.expect, r.ShouldIncludeBasedOnTag(tt.when))
		})
	}
}
func TestShouldIncludeBasedOnAdditionalTag(t *testing.T) {

	type arg struct {
		ExcludeRule        FilterRule
		ProtectUntilExpire bool
	}
	tests := []struct {
		name   string
		given  arg
		when   map[string]string
		expect bool
	}{
		{
			name:   "should include resource with default exclude tag",
			given:  arg{},
			when:   map[string]string{DefaultAwsResourceExclusionTagKey: "true"},
			expect: false,
		},
		{
			name: "should include resource with custom exclude additional tag",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"my-custom-skip-tag": {RE: *regexp.MustCompile("")},
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": "true"},
			expect: false,
		},
		{
			name: "should include resource with custom exclude additional tag and empty value (using regular expression)",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"my-custom-skip-tag": {RE: *regexp.MustCompile(".*")},
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": ""},
			expect: false,
		},
		{
			name: "should include resource with custom exclude tag and prefix value (using regular expression)",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"my-custom-skip-tag": {RE: *regexp.MustCompile("protected-.*")},
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-custom-skip-tag": "protected-database"},
			expect: false,
		},
		{
			name: "should include resource with custom multiple additional tag and prefix value (using regular expression)",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"my-first-skip-tag":  {RE: *regexp.MustCompile("protected-.*")},
						"my-second-skip-tag": {RE: *regexp.MustCompile("protected-.*")},
					},
				},
				ProtectUntilExpire: false,
			},
			when:   map[string]string{"my-second-skip-tag": "protected-database"},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ResourceType{
				ExcludeRule:        tt.given.ExcludeRule,
				ProtectUntilExpire: tt.given.ProtectUntilExpire,
			}

			require.Equal(t, tt.expect, r.ShouldIncludeBasedOnTag(tt.when))
		})
	}
}

func TestShouldIncludeWithTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want bool
	}{
		{
			name: "should include when resource has no tags",
			tags: map[string]string{},
			want: true,
		},
		{
			name: "should include when resource has tags",
			tags: map[string]string{"env": "production"},
			want: true,
		},
		{
			name: "should include when resource has default skip tag set",
			tags: map[string]string{DefaultAwsResourceExclusionTagKey: "true"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := &Config{
				ACM: ResourceType{},
			}

			assert.Equal(t, tt.want, testConfig.ACM.ShouldInclude(ResourceValue{
				Tags: tt.tags,
			}))
		})
	}
}
