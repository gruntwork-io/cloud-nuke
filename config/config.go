/*Package config - Todos
 * - validate that yaml conforms to the spec
 */
package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

// RawConfig - Struct unmarshalling the raw config file
type RawConfig struct {
	S3 resourceType `yaml:"s3"`
}

type resourceType struct {
	IncludeRule rawFilterRule `yaml:"include"`
	ExcludeRule rawFilterRule `yaml:"exclude"`
}

type rawFilterRule struct {
	NamesRE []string `yaml:"names_regex"`
}

// ConfigObj - Struct defining the config object we pass around
type ConfigObj struct {
	S3 Rules
}

// Rules - defines what to include and exclude
type Rules struct {
	IncludeRule FilterRule
	ExcludeRule FilterRule
}

// FilterRule - contains regular expressions or plain text patterns
type FilterRule struct {
	NamesRE []*regexp.Regexp
}

// GetConfig - unmarshalls the raw config file
// and parses it into a config object.
func GetConfig(filePath string) (ConfigObj, error) {
	var configObj ConfigObj

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return ConfigObj{}, err
	}

	yamlFile, err := ioutil.ReadFile(absolutePath)
	if err != nil {
		return ConfigObj{}, err
	}

	rawConfig := RawConfig{}

	err = yaml.Unmarshal(yamlFile, &rawConfig)
	if err != nil {
		return ConfigObj{}, err
	}

	for _, pattern := range rawConfig.S3.IncludeRule.NamesRE {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return ConfigObj{}, err
		}

		configObj.S3.IncludeRule.NamesRE = append(configObj.S3.IncludeRule.NamesRE, re)
	}

	for _, pattern := range rawConfig.S3.ExcludeRule.NamesRE {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return ConfigObj{}, err
		}

		configObj.S3.ExcludeRule.NamesRE = append(configObj.S3.ExcludeRule.NamesRE, re)
	}

	return configObj, nil
}
