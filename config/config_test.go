package config

import (
	"os"
	"path/filepath"
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
		ACM:                             ResourceType{},
		ACMPCA:                          ResourceType{},
		AMI:                             ResourceType{},
		APIGateway:                      ResourceType{},
		APIGatewayV2:                    ResourceType{},
		AccessAnalyzer:                  ResourceType{},
		AutoScalingGroup:                ResourceType{},
		AppRunnerService:                ResourceType{},
		BackupVault:                     ResourceType{},
		ManagedPrometheus:               ResourceType{},
		CloudWatchAlarm:                 ResourceType{},
		CloudWatchDashboard:             ResourceType{},
		CloudWatchLogGroup:              ResourceType{},
		CloudTrailTrail:                 ResourceType{},
		CloudFrontDistribution:          ResourceType{},
		CloudFormationStack:             ResourceType{},
		CodeDeployApplications:          ResourceType{},
		ConfigServiceRecorder:           ResourceType{},
		ConfigServiceRule:               ResourceType{},
		DataPipeline:                    ResourceType{},
		DataSyncLocation:                ResourceType{},
		DataSyncTask:                    ResourceType{},
		DBGlobalClusters:                ResourceType{},
		DBClusters:                      AWSProtectableResourceType{ResourceType: ResourceType{}},
		DBInstances:                     AWSProtectableResourceType{ResourceType: ResourceType{}},
		DBGlobalClusterMemberships:      ResourceType{},
		DBSubnetGroups:                  ResourceType{},
		DynamoDB:                        ResourceType{},
		EBSVolume:                       ResourceType{},
		ElasticBeanstalk:                ResourceType{},
		EC2:                             ResourceType{},
		EC2DedicatedHosts:               ResourceType{},
		EC2DHCPOption:                   ResourceType{},
		EC2KeyPairs:                     ResourceType{},
		EC2IPAM:                         ResourceType{},
		EC2IPAMPool:                     ResourceType{},
		EC2IPAMResourceDiscovery:        ResourceType{},
		EC2IPAMScope:                    ResourceType{},
		EC2Endpoint:                     EC2ResourceType{false, ResourceType{}},
		EC2Subnet:                       EC2ResourceType{false, ResourceType{}},
		EC2PlacementGroups:              ResourceType{},
		EgressOnlyInternetGateway:       ResourceType{},
		ECRRepository:                   ResourceType{},
		ECSCluster:                      ResourceType{},
		ECSService:                      ResourceType{},
		EKSCluster:                      ResourceType{},
		ELBv1:                           ResourceType{},
		ELBv2:                           ResourceType{},
		ElasticFileSystem:               ResourceType{},
		ElasticIP:                       ResourceType{},
		ElastiCache:                     ResourceType{},
		ElastiCacheParameterGroup:       ResourceType{},
		ElastiCacheServerless:           ResourceType{},
		ElastiCacheSubnetGroup:          ResourceType{},
		EventBridge:                     ResourceType{},
		EventBridgeArchive:              ResourceType{},
		EventBridgeRule:                 ResourceType{},
		EventBridgeSchedule:             ResourceType{},
		EventBridgeScheduleGroup:        ResourceType{},
		Grafana:                         ResourceType{},
		GuardDuty:                       ResourceType{},
		IAMGroups:                       ResourceType{},
		IAMPolicies:                     ResourceType{},
		IAMRoles:                        ResourceType{},
		IAMServiceLinkedRoles:           ResourceType{},
		IAMInstanceProfiles:             ResourceType{},
		IAMUsers:                        ResourceType{},
		KMSCustomerKeys:                 KMSCustomerKeyResourceType{false, ResourceType{}},
		KinesisStream:                   ResourceType{},
		KinesisFirehose:                 ResourceType{},
		LambdaFunction:                  ResourceType{},
		LambdaLayer:                     ResourceType{},
		LaunchConfiguration:             ResourceType{},
		LaunchTemplate:                  ResourceType{},
		MacieMember:                     ResourceType{},
		MSKCluster:                      ResourceType{},
		NATGateway:                      EC2ResourceType{false, ResourceType{}},
		OIDCProvider:                    ResourceType{},
		OpenSearchDomain:                ResourceType{},
		Redshift:                        ResourceType{},
		RDSSnapshot:                     ResourceType{},
		RDSParameterGroup:               ResourceType{},
		RDSProxy:                        ResourceType{},
		S3:                              ResourceType{},
		S3AccessPoint:                   ResourceType{},
		S3ObjectLambdaAccessPoint:       ResourceType{},
		S3MultiRegionAccessPoint:        ResourceType{},
		SESIdentity:                     ResourceType{},
		SESConfigurationSet:             ResourceType{},
		SESReceiptRuleSet:               ResourceType{},
		SESReceiptFilter:                ResourceType{},
		SESEmailTemplates:               ResourceType{},
		SNS:                             ResourceType{},
		SQS:                             ResourceType{},
		SageMakerNotebook:               ResourceType{},
		SageMakerStudioDomain:           ResourceType{},
		SecretsManager:                  ResourceType{},
		SecurityHub:                     ResourceType{},
		Snapshots:                       ResourceType{},
		TransitGateway:                  ResourceType{},
		TransitGatewayRouteTable:        ResourceType{},
		TransitGatewayVPCAttachment:     ResourceType{},
		TransitGatewayPeeringAttachment: ResourceType{},
		VPC:                             EC2ResourceType{false, ResourceType{}},
		Route53HostedZone:               ResourceType{},
		Route53CIDRCollection:           ResourceType{},
		Route53TrafficPolicy:            ResourceType{},
		InternetGateway:                 EC2ResourceType{false, ResourceType{}},
		NetworkACL:                      ResourceType{},
		NetworkInterface:                EC2ResourceType{false, ResourceType{}},
		SecurityGroup:                   EC2ResourceType{false, ResourceType{}},
		NetworkFirewall:                 ResourceType{},
		NetworkFirewallPolicy:           ResourceType{},
		NetworkFirewallRuleGroup:        ResourceType{},
		NetworkFirewallTLSConfig:        ResourceType{},
		NetworkFirewallResourcePolicy:   ResourceType{},
		VPCLatticeServiceNetwork:        ResourceType{},
		VPCLatticeService:               ResourceType{},
		VPCLatticeTargetGroup:           ResourceType{},
		RouteTable:                      EC2ResourceType{false, ResourceType{}},
		VPCPeeringConnection:            ResourceType{},

		// GCP Resources
		GCSBucket:        ResourceType{},
		CloudFunction:    ResourceType{},
		ArtifactRegistry: ResourceType{},
		GcpPubSubTopic:   ResourceType{},
	}
}

func TestNukeConfigFile(t *testing.T) {
	// Validate the production nuke config used by scheduled GitHub Actions
	configObj, err := GetConfig("../.github/nuke_config.yml")
	require.NoError(t, err, "nuke_config.yml should parse without errors")
	require.NotNil(t, configObj)

	// Guard against an accidentally emptied or truncated config file
	assert.NotEmpty(t, configObj.S3.ExcludeRule.NamesRegExp,
		"S3 exclude rules should be populated from nuke_config.yml")
}

func TestConfig_Garbage(t *testing.T) {
	configFilePath := "./mocks/garbage.yaml"
	_, err := GetConfig(configFilePath)
	require.Error(t, err)
}

func TestConfig_UnknownField(t *testing.T) {
	content := []byte("S3:\n  include:\n    names_regex:\n      - \".*\"\nBogusKey:\n  include:\n    names_regex:\n      - \".*\"\n")
	tmpFile := filepath.Join(t.TempDir(), "unknown.yaml")
	require.NoError(t, os.WriteFile(tmpFile, content, 0644))

	_, err := GetConfig(tmpFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "BogusKey")
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
	testConfig := &Config{}
	require.Equal(t, DefaultAwsResourceExclusionTagKey, testConfig.ACM.getExclusionTag())
}

func TestShouldIncludeBasedOnTag(t *testing.T) {
	timeIn2h := time.Now().Add(2 * time.Hour)

	type arg struct {
		ExcludeRule        FilterRule
		ProtectUntilExpire *bool
	}
	tests := []struct {
		name   string
		given  arg
		when   map[string]string
		expect bool
	}{
		{
			name:   "should exclude resource with default exclude tag set to true",
			given:  arg{},
			when:   map[string]string{DefaultAwsResourceExclusionTagKey: "true"},
			expect: false,
		},
		{
			name:   "should include resource when exclude tag is not set to true",
			given:  arg{},
			when:   map[string]string{DefaultAwsResourceExclusionTagKey: "false"},
			expect: true,
		},
		{
			name:   "should include resource when no tags are set",
			given:  arg{},
			when:   map[string]string{},
			expect: true,
		},
		{
			name: "should exclude resource when protection expires in the future (explicit true)",
			given: arg{
				ProtectUntilExpire: aws.Bool(true),
			},
			when:   map[string]string{CloudNukeAfterExclusionTagKey: timeIn2h.Format(time.RFC3339)},
			expect: false,
		},
		{
			name: "should exclude resource when protection expires in the future (nil defaults to true)",
			given: arg{
				ProtectUntilExpire: nil,
			},
			when:   map[string]string{CloudNukeAfterExclusionTagKey: timeIn2h.Format(time.RFC3339)},
			expect: false,
		},
		{
			name: "should include resource when protection expired in the past",
			given: arg{
				ProtectUntilExpire: aws.Bool(true),
			},
			when:   map[string]string{CloudNukeAfterExclusionTagKey: time.Now().Format(time.RFC3339)},
			expect: true,
		},
		{
			name: "should not protect resource when protect_until_expire is explicitly false",
			given: arg{
				ProtectUntilExpire: aws.Bool(false),
			},
			when:   map[string]string{CloudNukeAfterExclusionTagKey: timeIn2h.Format(time.RFC3339)},
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
		ProtectUntilExpire *bool
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
				ProtectUntilExpire: aws.Bool(false),
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
				ProtectUntilExpire: aws.Bool(false),
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
				ProtectUntilExpire: aws.Bool(false),
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
				ProtectUntilExpire: aws.Bool(false),
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

func TestShouldIncludeBasedOnTagLogic(t *testing.T) {
	type arg struct {
		IncludeRule        FilterRule
		ExcludeRule        FilterRule
		ProtectUntilExpire *bool
	}
	tests := []struct {
		name   string
		given  arg
		when   map[string]string
		expect bool
	}{
		{
			name: "should use OR logic by default for exclude tags",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team":    {RE: *regexp.MustCompile(".*")},
						"Service": {RE: *regexp.MustCompile(".*")},
					},
					// TagsOperator not specified, should default to OR
				},
			},
			when:   map[string]string{"Team": "backend", "env": "production"},
			expect: false, // Team tag matches, so should exclude (OR logic)
		},
		{
			name: "should use AND logic for exclude tags when specified",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team":    {RE: *regexp.MustCompile(".*")},
						"Service": {RE: *regexp.MustCompile(".*")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"Team": "backend", "env": "production"},
			expect: true, // Only Team tag matches, Service is missing, so should include (AND logic)
		},
		{
			name: "should exclude with AND logic when all tags match",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team":    {RE: *regexp.MustCompile(".*")},
						"Service": {RE: *regexp.MustCompile(".*")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"Team": "backend", "Service": "api", "env": "production"},
			expect: false, // Both Team and Service tags match, so should exclude (AND logic)
		},
		{
			name: "should work with case insensitive AND logic",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team":    {RE: *regexp.MustCompile(".*")},
						"Service": {RE: *regexp.MustCompile(".*")},
					},
					TagsOperator: "and", // lowercase should work
				},
			},
			when:   map[string]string{"Team": "backend", "Service": "api"},
			expect: false, // Both tags match with case insensitive AND
		},
		{
			name: "should use OR logic by default for include tags",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					// TagsOperator not specified, should default to OR
				},
			},
			when:   map[string]string{"env": "production", "Team": "frontend"},
			expect: true, // env matches production, so should include (OR logic)
		},
		{
			name: "should use AND logic for include tags when specified",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"env": "production", "Team": "frontend"},
			expect: false, // env matches but Team doesn't, so should exclude (AND logic)
		},
		{
			name: "should include with AND logic when all tags match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"env": "production", "Team": "backend"},
			expect: true, // Both env and Team match, so should include (AND logic)
		},
		{
			name: "tagging enforcement use case - exclude well-tagged resources",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team":    {RE: *regexp.MustCompile(".*")}, // Any value
						"Service": {RE: *regexp.MustCompile(".*")}, // Any value
					},
					TagsOperator: "AND", // Only exclude if BOTH tags are present
				},
			},
			when:   map[string]string{"Team": "backend", "env": "production"},
			expect: true, // Service tag missing, so include for destruction
		},
		{
			name: "tagging enforcement use case - keep well-tagged resources",
			given: arg{
				ExcludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team":    {RE: *regexp.MustCompile(".*")}, // Any value
						"Service": {RE: *regexp.MustCompile(".*")}, // Any value
					},
					TagsOperator: "AND", // Only exclude if BOTH tags are present
				},
			},
			when:   map[string]string{"Team": "backend", "Service": "api", "env": "production"},
			expect: false, // Both required tags present, so exclude (keep safe)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ResourceType{
				IncludeRule:        tt.given.IncludeRule,
				ExcludeRule:        tt.given.ExcludeRule,
				ProtectUntilExpire: tt.given.ProtectUntilExpire,
			}

			require.Equal(t, tt.expect, r.ShouldIncludeBasedOnTag(tt.when))
		})
	}
}

func TestShouldIncludeBasedOnIncludeRuleTags(t *testing.T) {
	type arg struct {
		IncludeRule FilterRule
	}
	tests := []struct {
		name   string
		given  arg
		when   map[string]string
		expect bool
	}{
		{
			name: "should include all resources when no include tags specified",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{},
				},
			},
			when:   map[string]string{"env": "production", "Team": "backend"},
			expect: true,
		},
		{
			name: "should include resource when single include tag matches",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env": {RE: *regexp.MustCompile("production")},
					},
				},
			},
			when:   map[string]string{"env": "production", "Team": "backend"},
			expect: true,
		},
		{
			name: "should exclude resource when single include tag doesn't match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env": {RE: *regexp.MustCompile("production")},
					},
				},
			},
			when:   map[string]string{"env": "staging", "Team": "backend"},
			expect: false,
		},
		{
			name: "should exclude resource when include tag key doesn't exist",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env": {RE: *regexp.MustCompile("production")},
					},
				},
			},
			when:   map[string]string{"Team": "backend"},
			expect: false,
		},
		{
			name: "should include resource with regex pattern matching",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env": {RE: *regexp.MustCompile("prod.*")},
					},
				},
			},
			when:   map[string]string{"env": "production", "Team": "backend"},
			expect: true,
		},
		{
			name: "should exclude resource when regex pattern doesn't match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env": {RE: *regexp.MustCompile("prod.*")},
					},
				},
			},
			when:   map[string]string{"env": "staging", "Team": "backend"},
			expect: false,
		},
		{
			name: "should include with OR logic when one of multiple tags matches (default)",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					// TagsOperator defaults to OR
				},
			},
			when:   map[string]string{"env": "staging", "Team": "backend"},
			expect: true, // Team matches, so include (OR logic)
		},
		{
			name: "should exclude with OR logic when none of multiple tags match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
				},
			},
			when:   map[string]string{"env": "staging", "Team": "frontend"},
			expect: false, // Neither tag matches, so exclude
		},
		{
			name: "should exclude with AND logic when not all tags match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"env": "production", "Team": "frontend"},
			expect: false, // env matches but Team doesn't, so exclude (AND logic)
		},
		{
			name: "should include with AND logic when all tags match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"env": "production", "Team": "backend"},
			expect: true, // Both tags match, so include (AND logic)
		},
		{
			name: "should exclude with AND logic when required tag is missing",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env":  {RE: *regexp.MustCompile("production")},
						"Team": {RE: *regexp.MustCompile("backend")},
					},
					TagsOperator: "AND",
				},
			},
			when:   map[string]string{"env": "production"},
			expect: false, // Team tag missing, so exclude (AND logic)
		},
		{
			name: "should work with case insensitive regex matching",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"env": {RE: *regexp.MustCompile("(?i)PRODUCTION")}, // case insensitive
					},
				},
			},
			when:   map[string]string{"env": "production"},
			expect: true,
		},
		{
			name: "should work with complex regex patterns",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team": {RE: *regexp.MustCompile("^(backend|frontend|devops)$")},
					},
				},
			},
			when:   map[string]string{"Team": "backend"},
			expect: true,
		},
		{
			name: "should exclude with complex regex when pattern doesn't match",
			given: arg{
				IncludeRule: FilterRule{
					Tags: map[string]Expression{
						"Team": {RE: *regexp.MustCompile("^(backend|frontend|devops)$")},
					},
				},
			},
			when:   map[string]string{"Team": "qa"},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ResourceType{
				IncludeRule: tt.given.IncludeRule,
			}

			require.Equal(t, tt.expect, r.ShouldIncludeBasedOnTag(tt.when))
		})
	}
}

func TestAllResourceTypesComplete(t *testing.T) {
	// This test uses reflection to verify that allResourceTypes() covers every
	// struct field in Config. If you add a new field to Config, add it to
	// allResourceTypes() — this test will fail until you do.
	c := &Config{}
	got := c.allResourceTypes()

	// Build a set of pointers returned by allResourceTypes()
	ptrSet := make(map[uintptr]bool, len(got))
	for _, rt := range got {
		ptrSet[reflect.ValueOf(rt).Pointer()] = true
	}

	v := reflect.ValueOf(c).Elem()
	t.Logf("Config has %d fields, allResourceTypes returned %d pointers", v.NumField(), len(got))

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name

		// Find the embedded ResourceType within this field
		var rtPtr uintptr
		switch field.Type() {
		case reflect.TypeOf(ResourceType{}):
			rtPtr = field.Addr().Pointer()
		case reflect.TypeOf(EC2ResourceType{}):
			rtPtr = field.FieldByName("ResourceType").Addr().Pointer()
		case reflect.TypeOf(AWSProtectableResourceType{}):
			rtPtr = field.FieldByName("ResourceType").Addr().Pointer()
		case reflect.TypeOf(KMSCustomerKeyResourceType{}):
			rtPtr = field.FieldByName("ResourceType").Addr().Pointer()
		default:
			t.Fatalf("Config field %q has unexpected type %s", fieldName, field.Type())
		}

		if !ptrSet[rtPtr] {
			t.Errorf("Config field %q is missing from allResourceTypes()", fieldName)
		}
	}
}

func TestAllEC2ResourceTypesComplete(t *testing.T) {
	c := &Config{}
	got := c.allEC2ResourceTypes()

	ptrSet := make(map[uintptr]bool, len(got))
	for _, rt := range got {
		ptrSet[reflect.ValueOf(rt).Pointer()] = true
	}

	v := reflect.ValueOf(c).Elem()
	ec2Type := reflect.TypeOf(EC2ResourceType{})
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name
		if field.Type() != ec2Type {
			continue
		}
		if !ptrSet[field.Addr().Pointer()] {
			t.Errorf("EC2ResourceType field %q is missing from allEC2ResourceTypes()", fieldName)
		}
	}
}

func TestShouldIncludeBasedOnTag_NilTagsSafety(t *testing.T) {
	// When include tag filters are specified, resources that don't support tags (nil) should be excluded
	r := ResourceType{
		IncludeRule: FilterRule{
			Tags: map[string]Expression{
				"Environment": {RE: *regexp.MustCompile("production")},
			},
		},
	}

	// Resource doesn't support tags (nil) - should exclude for safety
	assert.False(t, r.ShouldIncludeBasedOnTag(nil))

	// Resource supports tags but has none (empty map) - should exclude because tags don't match
	assert.False(t, r.ShouldIncludeBasedOnTag(map[string]string{}))

	// Resource has matching tags - should include
	assert.True(t, r.ShouldIncludeBasedOnTag(map[string]string{"Environment": "production"}))

	// When no include tag filters specified, resource without tag support should be included
	r2 := ResourceType{}
	assert.True(t, r2.ShouldIncludeBasedOnTag(nil))
}
