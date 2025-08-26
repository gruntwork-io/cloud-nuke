package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
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
	ACM                             ResourceType                  `yaml:"ACM"`
	ACMPCA                          ResourceType                  `yaml:"ACMPCA"`
	AMI                             ResourceType                  `yaml:"AMI"`
	APIGateway                      ResourceType                  `yaml:"APIGateway"`
	APIGatewayV2                    ResourceType                  `yaml:"APIGatewayV2"`
	AccessAnalyzer                  ResourceType                  `yaml:"AccessAnalyzer"`
	AutoScalingGroup                ResourceType                  `yaml:"AutoScalingGroup"`
	AppRunnerService                ResourceType                  `yaml:"AppRunnerService"`
	BackupVault                     ResourceType                  `yaml:"BackupVault"`
	ManagedPrometheus               ResourceType                  `yaml:"ManagedPrometheus"`
	CloudWatchAlarm                 ResourceType                  `yaml:"CloudWatchAlarm"`
	CloudWatchDashboard             ResourceType                  `yaml:"CloudWatchDashboard"`
	CloudWatchLogGroup              ResourceType                  `yaml:"CloudWatchLogGroup"`
	CloudMapNamespace               ResourceType                  `yaml:"CloudMapNamespace"`
	CloudMapService                 ResourceType                  `yaml:"CloudMapService"`
	CloudtrailTrail                 ResourceType                  `yaml:"CloudtrailTrail"`
	CloudfrontDistribution          ResourceType                  `yaml:"CloudfrontDistribution"`
	CodeDeployApplications          ResourceType                  `yaml:"CodeDeployApplications"`
	ConfigServiceRecorder           ResourceType                  `yaml:"ConfigServiceRecorder"`
	ConfigServiceRule               ResourceType                  `yaml:"ConfigServiceRule"`
	DataSyncLocation                ResourceType                  `yaml:"DataSyncLocation"`
	DataSyncTask                    ResourceType                  `yaml:"DataSyncTask"`
	DBGlobalClusters                ResourceType                  `yaml:"DBGlobalClusters"`
	DBClusters                      ResourceType                  `yaml:"DBClusters"`
	DBInstances                     AWSProtectectableResourceType `yaml:"DBInstances"`
	DBGlobalClusterMemberships      ResourceType                  `yaml:"DBGlobalClusterMemberships"`
	DBSubnetGroups                  ResourceType                  `yaml:"DBSubnetGroups"`
	DynamoDB                        ResourceType                  `yaml:"DynamoDB"`
	EBSVolume                       ResourceType                  `yaml:"EBSVolume"`
	ElasticBeanstalk                ResourceType                  `yaml:"ElasticBeanstalk"`
	EC2                             ResourceType                  `yaml:"EC2"`
	EC2DedicatedHosts               ResourceType                  `yaml:"EC2DedicatedHosts"`
	EC2DHCPOption                   ResourceType                  `yaml:"EC2DhcpOption"`
	EC2KeyPairs                     ResourceType                  `yaml:"EC2KeyPairs"`
	EC2IPAM                         ResourceType                  `yaml:"EC2IPAM"`
	EC2IPAMPool                     ResourceType                  `yaml:"EC2IPAMPool"`
	EC2IPAMResourceDiscovery        ResourceType                  `yaml:"EC2IPAMResourceDiscovery"`
	EC2IPAMScope                    ResourceType                  `yaml:"EC2IPAMScope"`
	EC2Endpoint                     ResourceType                  `yaml:"EC2Endpoint"`
	EC2Subnet                       EC2ResourceType               `yaml:"EC2Subnet"`
	EC2PlacementGroups              ResourceType                  `yaml:"EC2PlacementGroups"`
	EgressOnlyInternetGateway       ResourceType                  `yaml:"EgressOnlyInternetGateway"`
	ECRRepository                   ResourceType                  `yaml:"ECRRepository"`
	ECSCluster                      ResourceType                  `yaml:"ECSCluster"`
	ECSService                      ResourceType                  `yaml:"ECSService"`
	EKSCluster                      ResourceType                  `yaml:"EKSCluster"`
	ELBv1                           ResourceType                  `yaml:"ELBv1"`
	ELBv2                           ResourceType                  `yaml:"ELBv2"`
	ElasticFileSystem               ResourceType                  `yaml:"ElasticFileSystem"`
	ElasticIP                       ResourceType                  `yaml:"ElasticIP"`
	Elasticache                     ResourceType                  `yaml:"Elasticache"`
	ElasticacheParameterGroups      ResourceType                  `yaml:"ElasticacheParameterGroups"`
	ElasticCacheServerless          ResourceType                  `yaml:"ElasticCacheServerless"`
	ElasticacheSubnetGroups         ResourceType                  `yaml:"ElasticacheSubnetGroups"`
	EventBridge                     ResourceType                  `yaml:"EventBridge"`
	EventBridgeArchive              ResourceType                  `yaml:"EventBridgeArchive"`
	EventBridgeRule                 ResourceType                  `yaml:"EventBridgeRule"`
	EventBridgeSchedule             ResourceType                  `yaml:"EventBridgeSchedule"`
	EventBridgeScheduleGroup        ResourceType                  `yaml:"EventBridgeScheduleGroup"`
	Grafana                         ResourceType                  `yaml:"Grafana"`
	GuardDuty                       ResourceType                  `yaml:"GuardDuty"`
	IAMGroups                       ResourceType                  `yaml:"IAMGroups"`
	IAMPolicies                     ResourceType                  `yaml:"IAMPolicies"`
	IAMInstanceProfiles             ResourceType                  `yaml:"IAMInstanceProfiles"`
	IAMRoles                        ResourceType                  `yaml:"IAMRoles"`
	IAMServiceLinkedRoles           ResourceType                  `yaml:"IAMServiceLinkedRoles"`
	IAMUsers                        ResourceType                  `yaml:"IAMUsers"`
	KMSCustomerKeys                 KMSCustomerKeyResourceType    `yaml:"KMSCustomerKeys"`
	KinesisStream                   ResourceType                  `yaml:"KinesisStream"`
	KinesisFirehose                 ResourceType                  `yaml:"KinesisFirehose"`
	LambdaFunction                  ResourceType                  `yaml:"LambdaFunction"`
	LambdaLayer                     ResourceType                  `yaml:"LambdaLayer"`
	LaunchConfiguration             ResourceType                  `yaml:"LaunchConfiguration"`
	LaunchTemplate                  ResourceType                  `yaml:"LaunchTemplate"`
	MacieMember                     ResourceType                  `yaml:"MacieMember"`
	MSKCluster                      ResourceType                  `yaml:"MSKCluster"`
	NatGateway                      ResourceType                  `yaml:"NatGateway"`
	OIDCProvider                    ResourceType                  `yaml:"OIDCProvider"`
	OpenSearchDomain                ResourceType                  `yaml:"OpenSearchDomain"`
	Redshift                        ResourceType                  `yaml:"Redshift"`
	RdsSnapshot                     ResourceType                  `yaml:"RdsSnapshot"`
	RdsParameterGroup               ResourceType                  `yaml:"RdsParameterGroup"`
	RdsProxy                        ResourceType                  `yaml:"RdsProxy"`
	S3                              ResourceType                  `yaml:"s3"`
	S3AccessPoint                   ResourceType                  `yaml:"S3AccessPoint"`
	S3ObjectLambdaAccessPoint       ResourceType                  `yaml:"S3ObjectLambdaAccessPoint"`
	S3MultiRegionAccessPoint        ResourceType                  `yaml:"S3MultiRegionAccessPoint"`
	SESIdentity                     ResourceType                  `yaml:"SesIdentity"`
	SESConfigurationSet             ResourceType                  `yaml:"SesConfigurationset"`
	SESReceiptRuleSet               ResourceType                  `yaml:"SesReceiptRuleSet"`
	SESReceiptFilter                ResourceType                  `yaml:"SesReceiptFilter"`
	SESEmailTemplates               ResourceType                  `yaml:"SesEmailTemplates"`
	SNS                             ResourceType                  `yaml:"SNS"`
	SQS                             ResourceType                  `yaml:"SQS"`
	SageMakerEndpoint               ResourceType                  `yaml:"SageMakerEndpoint"`
	SageMakerEndpointConfig         ResourceType                  `yaml:"SageMakerEndpointConfig"`
	SageMakerNotebook               ResourceType                  `yaml:"SageMakerNotebook"`
	SageMakerStudioDomain           ResourceType                  `yaml:"SageMakerStudioDomain"`
	SecretsManagerSecrets           ResourceType                  `yaml:"SecretsManager"`
	SecurityHub                     ResourceType                  `yaml:"SecurityHub"`
	Snapshots                       ResourceType                  `yaml:"Snapshots"`
	TransitGateway                  ResourceType                  `yaml:"TransitGateway"`
	TransitGatewayRouteTable        ResourceType                  `yaml:"TransitGatewayRouteTable"`
	TransitGatewaysVpcAttachment    ResourceType                  `yaml:"TransitGatewaysVpcAttachment"`
	TransitGatewayPeeringAttachment ResourceType                  `yaml:"TransitGatewayPeeringAttachment"`
	VPC                             EC2ResourceType               `yaml:"VPC"`
	Route53HostedZone               ResourceType                  `yaml:"Route53HostedZone"`
	Route53CIDRCollection           ResourceType                  `yaml:"Route53CIDRCollection"`
	Route53TrafficPolicy            ResourceType                  `yaml:"Route53TrafficPolicy"`
	InternetGateway                 ResourceType                  `yaml:"InternetGateway"`
	NetworkACL                      ResourceType                  `yaml:"NetworkACL"`
	NetworkInterface                ResourceType                  `yaml:"NetworkInterface"`
	SecurityGroup                   EC2ResourceType               `yaml:"SecurityGroup"`
	NetworkFirewall                 ResourceType                  `yaml:"NetworkFirewall"`
	NetworkFirewallPolicy           ResourceType                  `yaml:"NetworkFirewallPolicy"`
	NetworkFirewallRuleGroup        ResourceType                  `yaml:"NetworkFirewallRuleGroup"`
	NetworkFirewallTLSConfig        ResourceType                  `yaml:"NetworkFirewallTLSConfig"`
	NetworkFirewallResourcePolicy   ResourceType                  `yaml:"NetworkFirewallResourcePolicy"`
	VPCLatticeServiceNetwork        ResourceType                  `yaml:"VPCLatticeServiceNetwork"`
	VPCLatticeService               ResourceType                  `yaml:"VPCLatticeService"`
	VPCLatticeTargetGroup           ResourceType                  `yaml:"VPCLatticeTargetGroup"`

	// GCP Resources
	GCSBucket ResourceType `yaml:"GCSBucket"`
}

func (c *Config) addTimeAfterFilter(timeFilter *time.Time, fieldName string) {
	// Do nothing if the time filter is nil
	if timeFilter == nil {
		return
	}

	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Struct {
			continue
		}

		ruleField := field.FieldByName(fieldName)
		filterRule := ruleField.Addr().Interface().(*FilterRule)
		filterRule.TimeAfter = timeFilter
	}
}
func (c *Config) addTimeOut(timeout *time.Duration, fieldName string) {
	// Do nothing if the time filter is nil or 0s
	if timeout == nil || *timeout <= 0 {
		return
	}

	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Struct {
			continue
		}

		timeoutField := field.FieldByName(fieldName)
		timeoutVal := timeoutField.Addr().Interface().(*string)
		*timeoutVal = timeout.String()
	}
}

func (c *Config) addDefautlOnly(flag bool) {
	// Do nothing if the flag filter is false, by default it will be false
	if flag == false {
		return
	}

	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Struct {
			continue
		}

		defaultOnlyField := field.FieldByName("DefaultOnly")
		// IsValid reports whether v represents a value.
		// It returns false if v is the zero Value.
		// If IsValid returns false, all other methods except String panic.
		if defaultOnlyField.IsValid() {
			defaultOnlyVal := defaultOnlyField.Addr().Interface().(*bool)
			*defaultOnlyVal = flag
		}
	}
}

func (c *Config) addBoolFlag(flag bool, fieldName string) {
	// Do nothing if the flag filter is false, by default it will be false
	if flag == false {
		return
	}

	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Struct {
			continue
		}

		defaultOnlyField := field.FieldByName(fieldName)
		// IsValid reports whether v represents a value.
		// It returns false if v is the zero Value.
		// If IsValid returns false, all other methods except String panic.
		if defaultOnlyField.IsValid() {
			defaultOnlyVal := defaultOnlyField.Addr().Interface().(*bool)
			*defaultOnlyVal = flag
		}
	}
}

func (c *Config) AddIncludeAfterTime(includeAfter *time.Time) {
	// include after filter has been applied to all resources via `newer-than` flag, we are
	// setting this rule across all resource types.
	c.addTimeAfterFilter(includeAfter, "IncludeRule")
}

func (c *Config) AddExcludeAfterTime(excludeAfter *time.Time) {
	// exclude after filter has been applied to all resources via `older-than` flag, we are
	// setting this rule across all resource types.
	c.addTimeAfterFilter(excludeAfter, "ExcludeRule")
}

func (c *Config) AddTimeout(timeout *time.Duration) {
	// timeout filter has been applied to all resources via `timeout` flag, we are
	// setting this rule across all resource types.
	c.addTimeOut(timeout, "Timeout")
}

func (c *Config) AddEC2DefaultOnly(flag bool) {
	// The flag filter has been applied to all resources via the default-only flag.
	// We are now setting this rule across all resource types that have a field named `DefaultOnly`.
	c.addBoolFlag(flag, "DefaultOnly")
}

func (c *Config) AddProtectUntilExpireFlag(flag bool) {
	// We are now setting this rule across all resource types that have a field named `ProtectUntilExpire`.
	c.addBoolFlag(flag, "ProtectUntilExpire")
}

type KMSCustomerKeyResourceType struct {
	IncludeUnaliasedKeys bool `yaml:"include_unaliased_keys"`
	ResourceType         `yaml:",inline"`
}

type EC2ResourceType struct {
	DefaultOnly  bool `yaml:"default_only"`
	ResourceType `yaml:",inline"`
}

type AWSProtectectableResourceType struct {
	ResourceType             `yaml:",inline"`
	IncludeDeletionProtected bool `yaml:"include_deletion_protected"`
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
	Tag          *string               `yaml:"tag"`       // Deprecated ~ A tag to filter resources by. (e.g., If set under ExcludedRule, resources with this tag will be excluded).
	TagValue     *Expression           `yaml:"tag_value"` // Deprecated
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

	yamlFile, err := ioutil.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, &configObj)
	if err != nil {
		return nil, err
	}

	return &configObj, nil
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
	if r.ExcludeRule.Tag != nil {
		return *r.ExcludeRule.Tag
	}

	return DefaultAwsResourceExclusionTagKey
}

func (r ResourceType) getExclusionTagValue() *Expression {
	if r.ExcludeRule.TagValue != nil {
		return r.ExcludeRule.TagValue
	}

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
