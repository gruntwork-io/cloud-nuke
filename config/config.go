package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"gopkg.in/yaml.v2"
)

const (
	DefaultAwsResourceExclusionTagKey   = "cloud-nuke-excluded"
	DefaultAwsResourceExclusionTagValue = "true"
	CloudNukeAfterExclusionTagKey       = "cloud-nuke-after"
	CloudNukeAfterTimeFormat            = time.RFC3339
	CloudNukeAfterTimeFormatLegacy      = time.DateTime
)

// Config - the config object we pass around
type Config struct {
	ACM                             ResourceType               `yaml:"ACM"`
	ACMPCA                          ResourceType               `yaml:"ACMPCA"`
	AMI                             ResourceType               `yaml:"AMI"`
	APIGateway                      ResourceType               `yaml:"APIGateway"`
	APIGatewayV2                    ResourceType               `yaml:"APIGatewayV2"`
	AccessAnalyzer                  ResourceType               `yaml:"AccessAnalyzer"`
	AutoScalingGroup                ResourceType               `yaml:"AutoScalingGroup"`
	AppRunnerService                ResourceType               `yaml:"AppRunnerService"`
	BackupVault                     ResourceType               `yaml:"BackupVault"`
	ManagedPrometheus               ResourceType               `yaml:"ManagedPrometheus"`
	CloudWatchAlarm                 ResourceType               `yaml:"CloudWatchAlarm"`
	CloudWatchDashboard             ResourceType               `yaml:"CloudWatchDashboard"`
	CloudWatchLogGroup              ResourceType               `yaml:"CloudWatchLogGroup"`
	CloudMapNamespace               ResourceType               `yaml:"CloudMapNamespace"`
	CloudMapService                 ResourceType               `yaml:"CloudMapService"`
	CloudTrailTrail                 ResourceType               `yaml:"CloudTrailTrail"`
	CloudFrontDistribution          ResourceType               `yaml:"CloudFrontDistribution"`
	CloudFormationStack             ResourceType               `yaml:"CloudFormationStack"`
	CodeDeployApplications          ResourceType               `yaml:"CodeDeployApplications"`
	ConfigServiceRecorder           ResourceType               `yaml:"ConfigServiceRecorder"`
	ConfigServiceRule               ResourceType               `yaml:"ConfigServiceRule"`
	DataPipeline                    ResourceType               `yaml:"DataPipeline"`
	DataSyncLocation                ResourceType               `yaml:"DataSyncLocation"`
	DataSyncTask                    ResourceType               `yaml:"DataSyncTask"`
	DBGlobalClusters                ResourceType               `yaml:"DBGlobalClusters"`
	DBClusters                      AWSProtectableResourceType `yaml:"DBClusters"`
	DBInstances                     AWSProtectableResourceType `yaml:"DBInstances"`
	DBGlobalClusterMemberships      ResourceType               `yaml:"DBGlobalClusterMemberships"`
	DBSubnetGroups                  ResourceType               `yaml:"DBSubnetGroups"`
	DynamoDB                        ResourceType               `yaml:"DynamoDB"`
	EBSVolume                       ResourceType               `yaml:"EBSVolume"`
	ElasticBeanstalk                ResourceType               `yaml:"ElasticBeanstalk"`
	EC2                             ResourceType               `yaml:"EC2"`
	EC2DedicatedHosts               ResourceType               `yaml:"EC2DedicatedHosts"`
	EC2DHCPOption                   ResourceType               `yaml:"EC2DHCPOption"`
	EC2KeyPairs                     ResourceType               `yaml:"EC2KeyPairs"`
	EC2IPAM                         ResourceType               `yaml:"EC2IPAM"`
	EC2IPAMByoasn                   ResourceType               `yaml:"EC2IPAMByoasn"`
	EC2IPAMCustomAllocation         ResourceType               `yaml:"EC2IPAMCustomAllocation"`
	EC2IPAMPool                     ResourceType               `yaml:"EC2IPAMPool"`
	EC2IPAMResourceDiscovery        ResourceType               `yaml:"EC2IPAMResourceDiscovery"`
	EC2IPAMScope                    ResourceType               `yaml:"EC2IPAMScope"`
	EC2Endpoint                     EC2ResourceType            `yaml:"EC2Endpoint"`
	EC2Subnet                       EC2ResourceType            `yaml:"EC2Subnet"`
	EC2PlacementGroups              ResourceType               `yaml:"EC2PlacementGroups"`
	EgressOnlyInternetGateway       ResourceType               `yaml:"EgressOnlyInternetGateway"`
	ECRRepository                   ResourceType               `yaml:"ECRRepository"`
	ECSCluster                      ResourceType               `yaml:"ECSCluster"`
	ECSService                      ResourceType               `yaml:"ECSService"`
	EKSCluster                      ResourceType               `yaml:"EKSCluster"`
	ELBv1                           ResourceType               `yaml:"ELBv1"`
	ELBv2                           ResourceType               `yaml:"ELBv2"`
	ElasticFileSystem               ResourceType               `yaml:"ElasticFileSystem"`
	ElasticIP                       ResourceType               `yaml:"ElasticIP"`
	ElastiCache                     ResourceType               `yaml:"ElastiCache"`
	ElastiCacheParameterGroup       ResourceType               `yaml:"ElastiCacheParameterGroup"`
	ElastiCacheServerless           ResourceType               `yaml:"ElastiCacheServerless"`
	ElastiCacheSubnetGroup          ResourceType               `yaml:"ElastiCacheSubnetGroup"`
	EventBridge                     ResourceType               `yaml:"EventBridge"`
	EventBridgeArchive              ResourceType               `yaml:"EventBridgeArchive"`
	EventBridgeRule                 ResourceType               `yaml:"EventBridgeRule"`
	EventBridgeSchedule             ResourceType               `yaml:"EventBridgeSchedule"`
	EventBridgeScheduleGroup        ResourceType               `yaml:"EventBridgeScheduleGroup"`
	Grafana                         ResourceType               `yaml:"Grafana"`
	GuardDuty                       ResourceType               `yaml:"GuardDuty"`
	IAMGroups                       ResourceType               `yaml:"IAMGroups"`
	IAMPolicies                     ResourceType               `yaml:"IAMPolicies"`
	IAMInstanceProfiles             ResourceType               `yaml:"IAMInstanceProfiles"`
	IAMRoles                        ResourceType               `yaml:"IAMRoles"`
	IAMServiceLinkedRoles           ResourceType               `yaml:"IAMServiceLinkedRoles"`
	IAMUsers                        ResourceType               `yaml:"IAMUsers"`
	KMSCustomerKeys                 KMSCustomerKeyResourceType `yaml:"KMSCustomerKeys"`
	KinesisStream                   ResourceType               `yaml:"KinesisStream"`
	KinesisFirehose                 ResourceType               `yaml:"KinesisFirehose"`
	LambdaFunction                  ResourceType               `yaml:"LambdaFunction"`
	LambdaLayer                     ResourceType               `yaml:"LambdaLayer"`
	LaunchConfiguration             ResourceType               `yaml:"LaunchConfiguration"`
	LaunchTemplate                  ResourceType               `yaml:"LaunchTemplate"`
	MacieMember                     ResourceType               `yaml:"MacieMember"`
	MSKCluster                      ResourceType               `yaml:"MSKCluster"`
	NATGateway                      EC2ResourceType            `yaml:"NATGateway"`
	OIDCProvider                    ResourceType               `yaml:"OIDCProvider"`
	OpenSearchDomain                ResourceType               `yaml:"OpenSearchDomain"`
	Redshift                        ResourceType               `yaml:"Redshift"`
	RedshiftSnapshotCopyGrant       ResourceType               `yaml:"RedshiftSnapshotCopyGrant"`
	RDSSnapshot                     ResourceType               `yaml:"RDSSnapshot"`
	RDSParameterGroup               ResourceType               `yaml:"RDSParameterGroup"`
	RDSProxy                        ResourceType               `yaml:"RDSProxy"`
	S3                              ResourceType               `yaml:"S3"`
	S3AccessPoint                   ResourceType               `yaml:"S3AccessPoint"`
	S3ObjectLambdaAccessPoint       ResourceType               `yaml:"S3ObjectLambdaAccessPoint"`
	S3MultiRegionAccessPoint        ResourceType               `yaml:"S3MultiRegionAccessPoint"`
	SESIdentity                     ResourceType               `yaml:"SESIdentity"`
	SESConfigurationSet             ResourceType               `yaml:"SESConfigurationSet"`
	SESReceiptRuleSet               ResourceType               `yaml:"SESReceiptRuleSet"`
	SESReceiptFilter                ResourceType               `yaml:"SESReceiptFilter"`
	SESEmailTemplates               ResourceType               `yaml:"SESEmailTemplates"`
	SNS                             ResourceType               `yaml:"SNS"`
	SQS                             ResourceType               `yaml:"SQS"`
	SageMakerEndpoint               ResourceType               `yaml:"SageMakerEndpoint"`
	SageMakerEndpointConfig         ResourceType               `yaml:"SageMakerEndpointConfig"`
	SageMakerNotebook               ResourceType               `yaml:"SageMakerNotebook"`
	SageMakerStudioDomain           ResourceType               `yaml:"SageMakerStudioDomain"`
	SecretsManager                  ResourceType               `yaml:"SecretsManager"`
	SecurityHub                     ResourceType               `yaml:"SecurityHub"`
	Snapshots                       ResourceType               `yaml:"Snapshots"`
	TransitGateway                  ResourceType               `yaml:"TransitGateway"`
	TransitGatewayRouteTable        ResourceType               `yaml:"TransitGatewayRouteTable"`
	TransitGatewayVPCAttachment     ResourceType               `yaml:"TransitGatewayVPCAttachment"`
	TransitGatewayPeeringAttachment ResourceType               `yaml:"TransitGatewayPeeringAttachment"`
	VPC                             EC2ResourceType            `yaml:"VPC"`
	Route53HostedZone               ResourceType               `yaml:"Route53HostedZone"`
	Route53CIDRCollection           ResourceType               `yaml:"Route53CIDRCollection"`
	Route53TrafficPolicy            ResourceType               `yaml:"Route53TrafficPolicy"`
	InternetGateway                 EC2ResourceType            `yaml:"InternetGateway"`
	NetworkACL                      ResourceType               `yaml:"NetworkACL"`
	NetworkInterface                EC2ResourceType            `yaml:"NetworkInterface"`
	SecurityGroup                   EC2ResourceType            `yaml:"SecurityGroup"`
	NetworkFirewall                 ResourceType               `yaml:"NetworkFirewall"`
	NetworkFirewallPolicy           ResourceType               `yaml:"NetworkFirewallPolicy"`
	NetworkFirewallRuleGroup        ResourceType               `yaml:"NetworkFirewallRuleGroup"`
	NetworkFirewallTLSConfig        ResourceType               `yaml:"NetworkFirewallTLSConfig"`
	NetworkFirewallResourcePolicy   ResourceType               `yaml:"NetworkFirewallResourcePolicy"`
	VPCLatticeServiceNetwork        ResourceType               `yaml:"VPCLatticeServiceNetwork"`
	VPCLatticeService               ResourceType               `yaml:"VPCLatticeService"`
	VPCLatticeTargetGroup           ResourceType               `yaml:"VPCLatticeTargetGroup"`
	RouteTable                      EC2ResourceType            `yaml:"RouteTable"`
	VPCPeeringConnection            ResourceType               `yaml:"VPCPeeringConnection"`

	// GCP Resources
	GCSBucket        ResourceType `yaml:"GCSBucket"`
	CloudFunction    ResourceType `yaml:"CloudFunction"`
	ArtifactRegistry ResourceType `yaml:"ArtifactRegistry"`
	GcpPubSubTopic     ResourceType `yaml:"GcpPubSubTopic"`
	GcpCloudRunService ResourceType `yaml:"GcpCloudRunService"`
}

// allResourceTypes returns pointers to the embedded ResourceType for every
// resource field in Config. This replaces the old reflection-based approach
// with a type-safe enumeration. If you add a new field to Config, add it here
// too — TestAllResourceTypesComplete will catch any omission.
func (c *Config) allResourceTypes() []*ResourceType {
	return []*ResourceType{
		&c.ACM,
		&c.ACMPCA,
		&c.AMI,
		&c.APIGateway,
		&c.APIGatewayV2,
		&c.AccessAnalyzer,
		&c.AutoScalingGroup,
		&c.AppRunnerService,
		&c.BackupVault,
		&c.ManagedPrometheus,
		&c.CloudWatchAlarm,
		&c.CloudWatchDashboard,
		&c.CloudWatchLogGroup,
		&c.CloudMapNamespace,
		&c.CloudMapService,
		&c.CloudTrailTrail,
		&c.CloudFrontDistribution,
		&c.CloudFormationStack,
		&c.CodeDeployApplications,
		&c.ConfigServiceRecorder,
		&c.ConfigServiceRule,
		&c.DataPipeline,
		&c.DataSyncLocation,
		&c.DataSyncTask,
		&c.DBGlobalClusters,
		&c.DBClusters.ResourceType,
		&c.DBInstances.ResourceType,
		&c.DBGlobalClusterMemberships,
		&c.DBSubnetGroups,
		&c.DynamoDB,
		&c.EBSVolume,
		&c.ElasticBeanstalk,
		&c.EC2,
		&c.EC2DedicatedHosts,
		&c.EC2DHCPOption,
		&c.EC2KeyPairs,
		&c.EC2IPAM,
		&c.EC2IPAMByoasn,
		&c.EC2IPAMCustomAllocation,
		&c.EC2IPAMPool,
		&c.EC2IPAMResourceDiscovery,
		&c.EC2IPAMScope,
		&c.EC2Endpoint.ResourceType,
		&c.EC2Subnet.ResourceType,
		&c.EC2PlacementGroups,
		&c.EgressOnlyInternetGateway,
		&c.ECRRepository,
		&c.ECSCluster,
		&c.ECSService,
		&c.EKSCluster,
		&c.ELBv1,
		&c.ELBv2,
		&c.ElasticFileSystem,
		&c.ElasticIP,
		&c.ElastiCache,
		&c.ElastiCacheParameterGroup,
		&c.ElastiCacheServerless,
		&c.ElastiCacheSubnetGroup,
		&c.EventBridge,
		&c.EventBridgeArchive,
		&c.EventBridgeRule,
		&c.EventBridgeSchedule,
		&c.EventBridgeScheduleGroup,
		&c.Grafana,
		&c.GuardDuty,
		&c.IAMGroups,
		&c.IAMPolicies,
		&c.IAMInstanceProfiles,
		&c.IAMRoles,
		&c.IAMServiceLinkedRoles,
		&c.IAMUsers,
		&c.KMSCustomerKeys.ResourceType,
		&c.KinesisStream,
		&c.KinesisFirehose,
		&c.LambdaFunction,
		&c.LambdaLayer,
		&c.LaunchConfiguration,
		&c.LaunchTemplate,
		&c.MacieMember,
		&c.MSKCluster,
		&c.NATGateway.ResourceType,
		&c.OIDCProvider,
		&c.OpenSearchDomain,
		&c.Redshift,
		&c.RedshiftSnapshotCopyGrant,
		&c.RDSSnapshot,
		&c.RDSParameterGroup,
		&c.RDSProxy,
		&c.S3,
		&c.S3AccessPoint,
		&c.S3ObjectLambdaAccessPoint,
		&c.S3MultiRegionAccessPoint,
		&c.SESIdentity,
		&c.SESConfigurationSet,
		&c.SESReceiptRuleSet,
		&c.SESReceiptFilter,
		&c.SESEmailTemplates,
		&c.SNS,
		&c.SQS,
		&c.SageMakerEndpoint,
		&c.SageMakerEndpointConfig,
		&c.SageMakerNotebook,
		&c.SageMakerStudioDomain,
		&c.SecretsManager,
		&c.SecurityHub,
		&c.Snapshots,
		&c.TransitGateway,
		&c.TransitGatewayRouteTable,
		&c.TransitGatewayVPCAttachment,
		&c.TransitGatewayPeeringAttachment,
		&c.VPC.ResourceType,
		&c.Route53HostedZone,
		&c.Route53CIDRCollection,
		&c.Route53TrafficPolicy,
		&c.InternetGateway.ResourceType,
		&c.NetworkACL,
		&c.NetworkInterface.ResourceType,
		&c.SecurityGroup.ResourceType,
		&c.NetworkFirewall,
		&c.NetworkFirewallPolicy,
		&c.NetworkFirewallRuleGroup,
		&c.NetworkFirewallTLSConfig,
		&c.NetworkFirewallResourcePolicy,
		&c.VPCLatticeServiceNetwork,
		&c.VPCLatticeService,
		&c.VPCLatticeTargetGroup,
		&c.RouteTable.ResourceType,
		&c.VPCPeeringConnection,
		&c.GCSBucket,
		&c.CloudFunction,
		&c.ArtifactRegistry,
		&c.GcpPubSubTopic,
		&c.GcpCloudRunService,
	}
}

// allEC2ResourceTypes returns pointers to the EC2ResourceType fields in Config.
// These are the only fields that have a DefaultOnly flag.
func (c *Config) allEC2ResourceTypes() []*EC2ResourceType {
	return []*EC2ResourceType{
		&c.EC2Endpoint,
		&c.EC2Subnet,
		&c.NATGateway,
		&c.VPC,
		&c.InternetGateway,
		&c.NetworkInterface,
		&c.SecurityGroup,
		&c.RouteTable,
	}
}

func (c *Config) AddIncludeAfterTime(includeAfter *time.Time) {
	if includeAfter == nil {
		return
	}
	for _, rt := range c.allResourceTypes() {
		rt.IncludeRule.TimeAfter = includeAfter
	}
}

func (c *Config) AddExcludeAfterTime(excludeAfter *time.Time) {
	if excludeAfter == nil {
		return
	}
	for _, rt := range c.allResourceTypes() {
		rt.ExcludeRule.TimeAfter = excludeAfter
	}
}

func (c *Config) AddTimeout(timeout *time.Duration) {
	if timeout == nil || *timeout <= 0 {
		return
	}
	for _, rt := range c.allResourceTypes() {
		rt.Timeout = timeout.String()
	}
}

func (c *Config) AddEC2DefaultOnly(flag bool) {
	if !flag {
		return
	}
	for _, rt := range c.allEC2ResourceTypes() {
		rt.DefaultOnly = flag
	}
}

func (c *Config) AddProtectUntilExpireFlag(flag bool) {
	if !flag {
		return
	}
	for _, rt := range c.allResourceTypes() {
		rt.ProtectUntilExpire = flag
	}
}

type KMSCustomerKeyResourceType struct {
	IncludeUnaliasedKeys bool `yaml:"include_unaliased_keys"`
	ResourceType         `yaml:",inline"`
}

type EC2ResourceType struct {
	DefaultOnly  bool `yaml:"default_only"`
	ResourceType `yaml:",inline"`
}

type AWSProtectableResourceType struct {
	ResourceType `yaml:",inline"`
}

type ResourceType struct {
	IncludeRule        FilterRule `yaml:"include"`
	ExcludeRule        FilterRule `yaml:"exclude"`
	Timeout            string     `yaml:"timeout"`
	ProtectUntilExpire bool       `yaml:"protect_until_expire"`
}

type FilterRule struct {
	NamesRegExp  []Expression          `yaml:"names_regex"`
	TimeAfter    *time.Time            `yaml:"time_after"`
	TimeBefore   *time.Time            `yaml:"time_before"`
	Tags         map[string]Expression `yaml:"tags"`
	TagsOperator string                `yaml:"tags_operator"` // "AND" or "OR" - defaults to "OR" for backward compatibility
}

type Expression struct {
	RE regexp.Regexp
}

// UnmarshalText - Internally used by yaml.Unmarshal to unmarshall an Expression field
func (expression *Expression) UnmarshalText(data []byte) error {
	var pattern string

	if err := yaml.Unmarshal(data, &pattern); err != nil {
		return err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	expression.RE = *re

	return nil
}

// GetConfig - Unmarshall the config file and parse it into a config object.
func GetConfig(filePath string) (*Config, error) {
	var configObj Config

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}

	err = yaml.UnmarshalStrict(yamlFile, &configObj)
	if err != nil {
		return nil, err
	}

	return &configObj, nil
}

// ApplyTimeFilters applies time-based filters to the config
func (c *Config) ApplyTimeFilters(excludeAfter, includeAfter *time.Time) {
	if excludeAfter != nil {
		c.AddExcludeAfterTime(excludeAfter)
	}
	if includeAfter != nil {
		c.AddIncludeAfterTime(includeAfter)
	}
}

func matches(name string, regexps []Expression) bool {
	for _, re := range regexps {
		if re.RE.MatchString(name) {
			return true
		}
	}
	return false
}

// matchesTags checks if the given tags match the tag expressions according to the specified logic (AND/OR)
func matchesTags(tags map[string]string, tagExpressions map[string]Expression, logic string) bool {
	// If no tag expressions are provided, no tags can match
	if len(tagExpressions) == 0 {
		return false
	}

	// Determine the logic to use - default to OR for backward compatibility
	useAndLogic := strings.ToUpper(logic) == "AND"
	if useAndLogic {
		return matchesTagsAnd(tags, tagExpressions)
	}

	return matchesTagsOr(tags, tagExpressions)
}

// matchesTagsAnd implements AND logic - all tag expressions must match for the function to return true
func matchesTagsAnd(tags map[string]string, tagExpressions map[string]Expression) bool {
	for tagKey, tagExpression := range tagExpressions {
		// Check if the tag key exists in the resource tags
		value, exists := tags[tagKey]
		if !exists {
			// If any required tag is missing, AND logic fails
			return false
		}
		// Check if the tag value matches the regex pattern (case-insensitive)
		if !tagExpression.RE.MatchString(strings.ToLower(value)) {
			// If any tag value doesn't match, AND logic fails
			return false
		}
	}
	// All tag expressions matched successfully
	return true
}

// matchesTagsOr implements OR logic - at least one tag expression must match for the function to return true
func matchesTagsOr(tags map[string]string, tagExpressions map[string]Expression) bool {
	for tagKey, tagExpression := range tagExpressions {
		// Check if the tag key exists in the resource tags
		value, exists := tags[tagKey]
		if !exists {
			// Skip this tag if it doesn't exist, continue checking others
			continue
		}
		// Check if the tag value matches the regex pattern (case-insensitive)
		if tagExpression.RE.MatchString(strings.ToLower(value)) {
			// If any tag matches, OR logic succeeds
			return true
		}
	}
	// No tag expressions matched
	return false
}

// ShouldInclude - Checks if a resource's Name should be included according to the inclusion and exclusion rules
func ShouldInclude(name *string, includeREs []Expression, excludeREs []Expression) bool {
	var resourceName string
	if name != nil {
		resourceName = *name
	}

	if len(includeREs) == 0 && len(excludeREs) == 0 {
		// If no rules are defined, should always include
		return true
	} else if matches(resourceName, excludeREs) {
		// If a rule that exclude matches, should not include
		return false
	} else if len(includeREs) == 0 {
		// Given the 'Name' is not in the 'exclude' list, should include if there is no 'include' list
		return true
	} else {
		// Given there is a 'include' list, and 'Name' is there, should include
		return matches(resourceName, includeREs)
	}
}

type ResourceValue struct {
	Name *string
	Time *time.Time
	Tags map[string]string
}

func (r ResourceType) ShouldIncludeBasedOnTime(time time.Time) bool {
	if r.ExcludeRule.TimeAfter != nil && time.After(*r.ExcludeRule.TimeAfter) {
		return false
	} else if r.ExcludeRule.TimeBefore != nil && time.Before(*r.ExcludeRule.TimeBefore) {
		return false
	} else if r.IncludeRule.TimeAfter != nil && time.Before(*r.IncludeRule.TimeAfter) {
		return false
	} else if r.IncludeRule.TimeBefore != nil && time.After(*r.IncludeRule.TimeBefore) {
		return false
	}

	return true
}

func (r ResourceType) getExclusionTag() string {
	return DefaultAwsResourceExclusionTagKey
}

func (r ResourceType) getExclusionTagValue() *Expression {
	return &Expression{RE: *regexp.MustCompile(DefaultAwsResourceExclusionTagValue)}
}

func ParseTimestamp(timestamp string) (*time.Time, error) {
	parsed, err := time.Parse(CloudNukeAfterTimeFormat, timestamp)
	if err != nil {
		logging.Debugf("Error parsing the timestamp into a `%v` Time format. Trying parsing the timestamp using the legacy `time.DateTime` format.", CloudNukeAfterTimeFormat)
		parsed, err = time.Parse(CloudNukeAfterTimeFormatLegacy, timestamp)
		if err != nil {
			logging.Debugf("Error parsing the timestamp into legacy `time.DateTime` Time format")
			return nil, err
		}
	}

	return &parsed, nil
}

func (r ResourceType) ShouldIncludeBasedOnTag(tags map[string]string) bool {
	// SAFETY CHECK: If include tag filters are specified but the resource doesn't support tags,
	// we should exclude it by default to prevent accidentally including unfiltered resources.
	// Resources that support tags always pass a map (even if empty), while resources that
	// don't support tags pass nil (by omitting the Tags field in ResourceValue).
	hasIncludeTagFilters := len(r.IncludeRule.Tags) > 0
	resourceDoesNotSupportTags := tags == nil

	if hasIncludeTagFilters && resourceDoesNotSupportTags {
		logging.Debugf("Resource does not support tag filtering but include tag filters are specified - excluding for safety")
		return false
	}

	// Handle exclude rule first
	exclusionTag := r.getExclusionTag()
	exclusionTagValue := r.getExclusionTagValue()
	if value, ok := tags[exclusionTag]; ok {
		if matches(strings.ToLower(value), []Expression{*exclusionTagValue}) {
			return false
		}
	}

	// Check additional exclude tags with AND/OR logic
	if matchesTags(tags, r.ExcludeRule.Tags, r.ExcludeRule.TagsOperator) {
		return false
	}

	if r.ProtectUntilExpire {
		// Check if the tags contain "cloud-nuke-after" and if the date is before today.
		if value, ok := tags[CloudNukeAfterExclusionTagKey]; ok {
			nukeDate, err := ParseTimestamp(value)
			if err == nil {
				if !nukeDate.Before(time.Now()) {
					logging.Debugf("[Skip] the resource is protected until %v", nukeDate)
					return false
				}
			}
		}
	}

	// Handle include rule with AND/OR logic
	if len(r.IncludeRule.Tags) > 0 {
		if !matchesTags(tags, r.IncludeRule.Tags, r.IncludeRule.TagsOperator) {
			return false
		}
	}

	return true
}

func (r ResourceType) ShouldInclude(value ResourceValue) bool {
	if !ShouldInclude(value.Name, r.IncludeRule.NamesRegExp, r.ExcludeRule.NamesRegExp) {
		return false
	} else if value.Time != nil && !r.ShouldIncludeBasedOnTime(*value.Time) {
		return false
	} else if !r.ShouldIncludeBasedOnTag(value.Tags) {
		return false
	}

	return true
}
