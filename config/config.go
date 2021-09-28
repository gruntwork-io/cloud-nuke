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
	// If no rules are defined, should always include
	if len(includeREs) == 0 && len(excludeREs) == 0 {
		return true
		// If a rule that exclude matches, should not include
	} else if matches(name, excludeREs) {
		return false
		// Given the 'name' is not in the 'exclude' list, should include if there is no 'include' list
	} else if len(includeREs) == 0 {
		return true
		// Given there is a 'include' list, and 'name' is there, should include
	} else if matches(name, includeREs) {
		return true
		// If it's not in the 'include' list, should not include
	} else {
		return false
	}
}
