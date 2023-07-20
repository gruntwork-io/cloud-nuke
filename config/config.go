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
	CloudtrailTrail              ResourceType               `yaml:"CloudtrailTrail"`
	CloudWatchAlarm              ResourceType               `yaml:"CloudWatchAlarm"`
	CloudWatchDashboard          ResourceType               `yaml:"CloudWatchDashboard"`
	CloudWatchLogGroup           ResourceType               `yaml:"CloudWatchLogGroup"`
	CodeDeployApplication        ResourceType               `yaml:"CodeDeployApplication"`
	ConfigServiceRecorder        ResourceType               `yaml:"ConfigServiceRecorder"`
	ConfigServiceRule            ResourceType               `yaml:"ConfigServiceRule"`
	DBCluster                    ResourceType               `yaml:"DBCluster"`
	DBInstance                   ResourceType               `yaml:"DBInstance"`
	DBSubnetGroup                ResourceType               `yaml:"DBSubnetGroup"`
	DynamoDB                     ResourceType               `yaml:"DynamoDB"`
	EBSVolume                    ResourceType               `yaml:"EBSVolume"`
	EC2DedicatedHost             ResourceType               `yaml:"EC2DedicatedHost"`
	EC2Instance                  ResourceType               `yaml:"EC2Instance"`
	EC2KeyPair                   ResourceType               `yaml:"EC2KeyPair"`
	EC2Snapshot                  ResourceType               `yaml:"EC2Snapshot"`
	EC2VPC                       ResourceType               `yaml:"EC2VPC"`
	ECR                          ResourceType               `yaml:"ECR"`
	ECSCluster                   ResourceType               `yaml:"ECSCluster"`
	ECSService                   ResourceType               `yaml:"ECSService"`
	EIPAddress                   ResourceType               `yaml:"EIPAddress"`
	EKSCluster                   ResourceType               `yaml:"EKSCluster"`
	ELBv1                        ResourceType               `yaml:"ELBv1"`
	ELBv2                        ResourceType               `yaml:"ELBv2"`
	ElasticFileSystem            ResourceType               `yaml:"ElasticFileSystem"`
	ElasticIP                    ResourceType               `yaml:"ElasticIP"`
	Elasticache                  ResourceType               `yaml:"Elasticache"`
	ElasticacheParameterGroup    ResourceType               `yaml:"ElasticacheParameterGroup"`
	ElasticacheSubnetGroup       ResourceType               `yaml:"ElasticacheSubnetGroup"`
	GuardDuty                    ResourceType               `yaml:"GuardDuty"`
	IAMGroup                     ResourceType               `yaml:"IAMGroup"`
	IAMPolicy                    ResourceType               `yaml:"IAMPolicy"`
	IAMRole                      ResourceType               `yaml:"IAMRole"`
	IAMServiceLinkedRole         ResourceType               `yaml:"IAMServiceLinkedRole"`
	IAMUser                      ResourceType               `yaml:"IAMUser"`
	KMSCustomerKey               KMSCustomerKeyResourceType `yaml:"KMSCustomerKey"`
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
	SQS                          ResourceType               `yaml:"SQS"`
	SageMakerNotebookInstance    ResourceType               `yaml:"SageMakerNotebookInstance"`
	SecretsManagerSecret         ResourceType               `yaml:"SecretsManager"`
	SecurityHub                  ResourceType               `yaml:"SecurityHub"`
	TransitGateway               ResourceType               `yaml:"TransitGateway"`
	TransitGatewayRouteTable     ResourceType               `yaml:"TransitGatewayRouteTable"`
	TransitGatewaysVpcAttachment ResourceType               `yaml:"TransitGatewaysVpcAttachment"`
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
