package config

import (
	"io/ioutil"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
)

// Config - the config object we pass around
type Config struct {
	S3                    ResourceType `yaml:"s3"`
	IAMUsers              ResourceType `yaml:"IAMUsers"`
	SecretsManagerSecrets ResourceType `yaml:"SecretsManager"`
	NatGateway            ResourceType `yaml:"NatGateway"`
	AccessAnalyzer        ResourceType `yaml:"AccessAnalyzer"`
	CloudWatchDashboard   ResourceType `yaml:"CloudWatchDashboard"`
	OpenSearchDomain      ResourceType `yaml:"OpenSearchDomain"`
	DynamoDB              ResourceType `yaml:"DynamoDB"`
	EBSVolume             ResourceType `yaml:"EBSVolume"`
	EFSInstances          ResourceType `yaml:"EFSInstances"`
	LambdaFunction        ResourceType `yaml:"LambdaFunction"`
	ELBv2                 ResourceType `yaml:"ELBv2"`
	ECSService            ResourceType `yaml:"ECSService"`
	ECSCluster            ResourceType `yaml:"ECSCluster"`
	Elasticache           ResourceType `yaml:"Elasticache"`
	VPC                   ResourceType `yaml:"VPC"`
	OIDCProvider          ResourceType `yaml:"OIDCProvider"`
	AutoScalingGroup      ResourceType `yaml:"AutoScalingGroup"`
	LaunchConfiguration   ResourceType `yaml:"LaunchConfiguration"`
	ElasticIP             ResourceType `yaml:"ElasticIP"`
	EC2                   ResourceType `yaml:"EC2"`
	CloudWatchLogGroup    ResourceType `yaml:"CloudWatchLogGroup"`
	KMSCustomerKeys       ResourceType `yaml:"KMSCustomerKeys"`
	EKSCluster            ResourceType `yaml:"EKSCluster"`
}

type ResourceType struct {
	IncludeRule FilterRule `yaml:"include"`
	ExcludeRule FilterRule `yaml:"exclude"`
}

type FilterRule struct {
	NamesRegExp []Expression `yaml:"names_regex"`
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

// ShouldInclude - Checks if a resource's name should be included according to the inclusion and exclusion rules
func ShouldInclude(name string, includeREs []Expression, excludeREs []Expression) bool {
	if len(includeREs) == 0 && len(excludeREs) == 0 {
		// If no rules are defined, should always include
		return true
	} else if matches(name, excludeREs) {
		// If a rule that exclude matches, should not include
		return false
	} else if len(includeREs) == 0 {
		// Given the 'name' is not in the 'exclude' list, should include if there is no 'include' list
		return true
	} else {
		// Given there is a 'include' list, and 'name' is there, should include
		return matches(name, includeREs)
	}
}
