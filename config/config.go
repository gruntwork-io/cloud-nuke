package config

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v2"
)

// Config - the config object we pass around
type Config struct {
	ACM                          ResourceType               `yaml:"ACM"`
	ACMPCA                       ResourceType               `yaml:"ACMPCA"`
	AMI                          ResourceType               `yaml:"AMI"`
	APIGateway                   ResourceType               `yaml:"APIGateway"`
	APIGatewayV2                 ResourceType               `yaml:"APIGatewayV2"`
	AccessAnalyzer               ResourceType               `yaml:"AccessAnalyzer"`
	AutoScalingGroup             ResourceType               `yaml:"AutoScalingGroup"`
	BackupVault                  ResourceType               `yaml:"BackupVault"`
	CloudWatchAlarm              ResourceType               `yaml:"CloudWatchAlarm"`
	CloudWatchDashboard          ResourceType               `yaml:"CloudWatchDashboard"`
	CloudWatchLogGroup           ResourceType               `yaml:"CloudWatchLogGroup"`
	CloudtrailTrail              ResourceType               `yaml:"CloudtrailTrail"`
	CodeDeployApplications       ResourceType               `yaml:"CodeDeployApplications"`
	ConfigServiceRecorder        ResourceType               `yaml:"ConfigServiceRecorder"`
	ConfigServiceRule            ResourceType               `yaml:"ConfigServiceRule"`
	DBClusters                   ResourceType               `yaml:"DBClusters"`
	DBInstances                  ResourceType               `yaml:"DBInstances"`
	DBSubnetGroups               ResourceType               `yaml:"DBSubnetGroups"`
	DynamoDB                     ResourceType               `yaml:"DynamoDB"`
	EBSVolume                    ResourceType               `yaml:"EBSVolume"`
	EC2                          ResourceType               `yaml:"EC2"`
	EC2DedicatedHosts            ResourceType               `yaml:"EC2DedicatedHosts"`
	EC2KeyPairs                  ResourceType               `yaml:"EC2KeyPairs"`
	ECRRepository                ResourceType               `yaml:"ECRRepository"`
	ECSCluster                   ResourceType               `yaml:"ECSCluster"`
	ECSService                   ResourceType               `yaml:"ECSService"`
	EKSCluster                   ResourceType               `yaml:"EKSCluster"`
	ELBv1                        ResourceType               `yaml:"ELBv1"`
	ELBv2                        ResourceType               `yaml:"ELBv2"`
	ElasticFileSystem            ResourceType               `yaml:"ElasticFileSystem"`
	ElasticIP                    ResourceType               `yaml:"ElasticIP"`
	Elasticache                  ResourceType               `yaml:"Elasticache"`
	ElasticacheParameterGroups   ResourceType               `yaml:"ElasticacheParameterGroups"`
	ElasticacheSubnetGroups      ResourceType               `yaml:"ElasticacheSubnetGroups"`
	GuardDuty                    ResourceType               `yaml:"GuardDuty"`
	IAMGroups                    ResourceType               `yaml:"IAMGroups"`
	IAMPolicies                  ResourceType               `yaml:"IAMPolicies"`
	IAMRoles                     ResourceType               `yaml:"IAMRoles"`
	IAMServiceLinkedRoles        ResourceType               `yaml:"IAMServiceLinkedRoles"`
	IAMUsers                     ResourceType               `yaml:"IAMUsers"`
	KMSCustomerKeys              KMSCustomerKeyResourceType `yaml:"KMSCustomerKeys"`
	KinesisStream                ResourceType               `yaml:"KinesisStream"`
	LambdaFunction               ResourceType               `yaml:"LambdaFunction"`
	LaunchConfiguration          ResourceType               `yaml:"LaunchConfiguration"`
	LaunchTemplate               ResourceType               `yaml:"LaunchTemplate"`
	MacieMember                  ResourceType               `yaml:"MacieMember"`
	NatGateway                   ResourceType               `yaml:"NatGateway"`
	OIDCProvider                 ResourceType               `yaml:"OIDCProvider"`
	OpenSearchDomain             ResourceType               `yaml:"OpenSearchDomain"`
	Redshift                     ResourceType               `yaml:"Redshift"`
	S3                           ResourceType               `yaml:"s3"`
	SNS                          ResourceType               `yaml:"SNS"`
	Snapshots                    ResourceType               `yaml:"Snapshots"`
	SQS                          ResourceType               `yaml:"SQS"`
	SageMakerNotebook            ResourceType               `yaml:"SageMakerNotebook"`
	SecretsManagerSecrets        ResourceType               `yaml:"SecretsManager"`
	SecurityHub                  ResourceType               `yaml:"SecurityHub"`
	TransitGateway               ResourceType               `yaml:"TransitGateway"`
	TransitGatewayRouteTable     ResourceType               `yaml:"TransitGatewayRouteTable"`
	TransitGatewaysVpcAttachment ResourceType               `yaml:"TransitGatewaysVpcAttachment"`
	VPC                          ResourceType               `yaml:"VPC"`
}

type KMSCustomerKeyResourceType struct {
	DeleteUnaliasedKeys bool `yaml:"delete_unaliased_keys"`

	ResourceType
}

type ResourceType struct {
	IncludeRule FilterRule `yaml:"include"`
	ExcludeRule FilterRule `yaml:"exclude"`
}

type FilterRule struct {
	NamesRegExp []Expression `yaml:"names_regex"`
	TimeAfter   *time.Time   `yaml:"time_after"`
	TimeBefore  *time.Time   `yaml:"time_before"`
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

func (r ResourceType) ShouldInclude(value ResourceValue) bool {
	if value.Name != nil && !ShouldInclude(*value.Name, r.IncludeRule.NamesRegExp, r.ExcludeRule.NamesRegExp) {
		return false
	} else if value.Time != nil && !r.ShouldIncludeBasedOnTime(*value.Time) {
		return false
	}

	return true
}
