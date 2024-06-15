package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const DefaultAwsResourceExclusionTagKey = "cloud-nuke-excluded"

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
	CloudWatchAlarm                 ResourceType               `yaml:"CloudWatchAlarm"`
	CloudWatchDashboard             ResourceType               `yaml:"CloudWatchDashboard"`
	CloudWatchLogGroup              ResourceType               `yaml:"CloudWatchLogGroup"`
	CloudtrailTrail                 ResourceType               `yaml:"CloudtrailTrail"`
	CodeDeployApplications          ResourceType               `yaml:"CodeDeployApplications"`
	ConfigServiceRecorder           ResourceType               `yaml:"ConfigServiceRecorder"`
	ConfigServiceRule               ResourceType               `yaml:"ConfigServiceRule"`
	DBGlobalClusters                ResourceType               `yaml:"DBGlobalClusters"`
	DBClusters                      ResourceType               `yaml:"DBClusters"`
	DBInstances                     ResourceType               `yaml:"DBInstances"`
	DBGlobalClusterMemberships      ResourceType               `yaml:"DBGlobalClusterMemberships"`
	DBSubnetGroups                  ResourceType               `yaml:"DBSubnetGroups"`
	DynamoDB                        ResourceType               `yaml:"DynamoDB"`
	EBSVolume                       ResourceType               `yaml:"EBSVolume"`
	ElasticBeanstalk                ResourceType               `yaml:"ElasticBeanstalk"`
	EC2                             ResourceType               `yaml:"EC2"`
	EC2DedicatedHosts               ResourceType               `yaml:"EC2DedicatedHosts"`
	EC2DHCPOption                   ResourceType               `yaml:"EC2DhcpOption"`
	EC2KeyPairs                     ResourceType               `yaml:"EC2KeyPairs"`
	EC2IPAM                         ResourceType               `yaml:"EC2IPAM"`
	EC2IPAMPool                     ResourceType               `yaml:"EC2IPAMPool"`
	EC2IPAMResourceDiscovery        ResourceType               `yaml:"EC2IPAMResourceDiscovery"`
	EC2IPAMScope                    ResourceType               `yaml:"EC2IPAMScope"`
	EC2Endpoint                     ResourceType               `yaml:"EC2Endpoint"`
	EC2Subnet                       EC2ResourceType            `yaml:"EC2Subnet"`
	EgressOnlyInternetGateway       ResourceType               `yaml:"EgressOnlyInternetGateway"`
	ECRRepository                   ResourceType               `yaml:"ECRRepository"`
	ECSCluster                      ResourceType               `yaml:"ECSCluster"`
	ECSService                      ResourceType               `yaml:"ECSService"`
	EKSCluster                      ResourceType               `yaml:"EKSCluster"`
	ELBv1                           ResourceType               `yaml:"ELBv1"`
	ELBv2                           ResourceType               `yaml:"ELBv2"`
	ElasticFileSystem               ResourceType               `yaml:"ElasticFileSystem"`
	ElasticIP                       ResourceType               `yaml:"ElasticIP"`
	Elasticache                     ResourceType               `yaml:"Elasticache"`
	ElasticacheParameterGroups      ResourceType               `yaml:"ElasticacheParameterGroups"`
	ElasticacheSubnetGroups         ResourceType               `yaml:"ElasticacheSubnetGroups"`
	GuardDuty                       ResourceType               `yaml:"GuardDuty"`
	IAMGroups                       ResourceType               `yaml:"IAMGroups"`
	IAMPolicies                     ResourceType               `yaml:"IAMPolicies"`
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
	NatGateway                      ResourceType               `yaml:"NatGateway"`
	OIDCProvider                    ResourceType               `yaml:"OIDCProvider"`
	OpenSearchDomain                ResourceType               `yaml:"OpenSearchDomain"`
	Redshift                        ResourceType               `yaml:"Redshift"`
	RdsSnapshot                     ResourceType               `yaml:"RdsSnapshot"`
	RdsParameterGroup               ResourceType               `yaml:"RdsParameterGroup"`
	RdsProxy                        ResourceType               `yaml:"RdsProxy"`
	S3                              ResourceType               `yaml:"s3"`
	S3AccessPoint                   ResourceType               `yaml:"S3AccessPoint"`
	S3ObjectLambdaAccessPoint       ResourceType               `yaml:"S3ObjectLambdaAccessPoint"`
	S3MultiRegionAccessPoint        ResourceType               `yaml:"S3MultiRegionAccessPoint"`
	SESIdentity                     ResourceType               `yaml:"SesIdentity"`
	SESConfigurationSet             ResourceType               `yaml:"SesConfigurationset"`
	SESReceiptRuleSet               ResourceType               `yaml:"SesReceiptRuleSet"`
	SESReceiptFilter                ResourceType               `yaml:"SesReceiptFilter"`
	SESEmailTemplates               ResourceType               `yaml:"SesEmailTemplates"`
	SNS                             ResourceType               `yaml:"SNS"`
	SQS                             ResourceType               `yaml:"SQS"`
	SageMakerNotebook               ResourceType               `yaml:"SageMakerNotebook"`
	SecretsManagerSecrets           ResourceType               `yaml:"SecretsManager"`
	SecurityHub                     ResourceType               `yaml:"SecurityHub"`
	Snapshots                       ResourceType               `yaml:"Snapshots"`
	TransitGateway                  ResourceType               `yaml:"TransitGateway"`
	TransitGatewayRouteTable        ResourceType               `yaml:"TransitGatewayRouteTable"`
	TransitGatewaysVpcAttachment    ResourceType               `yaml:"TransitGatewaysVpcAttachment"`
	TransitGatewayPeeringAttachment ResourceType               `yaml:"TransitGatewayPeeringAttachment"`
	VPC                             EC2ResourceType            `yaml:"VPC"`
	Route53HostedZone               ResourceType               `yaml:"Route53HostedZone"`
	Route53CIDRCollection           ResourceType               `yaml:"Route53CIDRCollection"`
	Route53TrafficPolicy            ResourceType               `yaml:"Route53TrafficPolicy"`
	InternetGateway                 ResourceType               `yaml:"InternetGateway"`
	NetworkACL                      ResourceType               `yaml:"NetworkACL"`
	NetworkInterface                ResourceType               `yaml:"NetworkInterface"`
	SecurityGroup                   EC2ResourceType            `yaml:"SecurityGroup"`
	NetworkFirewall                 ResourceType               `yaml:"NetworkFirewall"`
	NetworkFirewallPolicy           ResourceType               `yaml:"NetworkFirewallPolicy"`
	NetworkFirewallRuleGroup        ResourceType               `yaml:"NetworkFirewallRuleGroup"`
	NetworkFirewallTLSConfig        ResourceType               `yaml:"NetworkFirewallTLSConfig"`
	NetworkFirewallResourcePolicy   ResourceType               `yaml:"NetworkFirewallResourcePolicy"`
	VPCLatticeServiceNetwork        ResourceType               `yaml:"VPCLatticeServiceNetwork"`
	VPCLatticeService               ResourceType               `yaml:"VPCLatticeService"`
	VPCLatticeTargetGroup           ResourceType               `yaml:"VPCLatticeTargetGroup"`
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
	c.addDefautlOnly(flag)
}

type KMSCustomerKeyResourceType struct {
	IncludeUnaliasedKeys bool `yaml:"include_unaliased_keys"`
	ResourceType         `yaml:",inline"`
}
type EC2ResourceType struct {
	DefaultOnly  bool `yaml:"default_only"`
	ResourceType `yaml:",inline"`
}

type ResourceType struct {
	IncludeRule FilterRule `yaml:"include"`
	ExcludeRule FilterRule `yaml:"exclude"`
	Timeout     string     `yaml:"timeout"`
}

type FilterRule struct {
	NamesRegExp []Expression `yaml:"names_regex"`
	TimeAfter   *time.Time   `yaml:"time_after"`
	TimeBefore  *time.Time   `yaml:"time_before"`
	Tag         *string      `yaml:"tag"` // A tag to filter resources by. (e.g., If set under ExcludedRule, resources with this tag will be excluded).
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

// ShouldInclude - Checks if a resource's Name should be included according to the inclusion and exclusion rules
func ShouldInclude(name string, includeREs []Expression, excludeREs []Expression) bool {
	if len(includeREs) == 0 && len(excludeREs) == 0 {
		// If no rules are defined, should always include
		return true
	} else if matches(name, excludeREs) {
		// If a rule that exclude matches, should not include
		return false
	} else if len(includeREs) == 0 {
		// Given the 'Name' is not in the 'exclude' list, should include if there is no 'include' list
		return true
	} else {
		// Given there is a 'include' list, and 'Name' is there, should include
		return matches(name, includeREs)
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

func (r ResourceType) ShouldIncludeBasedOnTag(tags map[string]string) bool {
	// Handle exclude rule first
	exclusionTag := r.getExclusionTag()
	if value, ok := tags[exclusionTag]; ok {
		if strings.ToLower(value) == "true" {
			return false
		}
	}

	return true
}

func (r ResourceType) ShouldInclude(value ResourceValue) bool {
	if value.Name != nil && !ShouldInclude(*value.Name, r.IncludeRule.NamesRegExp, r.ExcludeRule.NamesRegExp) {
		return false
	} else if value.Time != nil && !r.ShouldIncludeBasedOnTime(*value.Time) {
		return false
	} else if value.Tags != nil && len(value.Tags) != 0 && !r.ShouldIncludeBasedOnTag(value.Tags) {
		return false
	}

	return true
}
