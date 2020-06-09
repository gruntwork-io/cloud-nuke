/* validate that yaml conforms to the spec
* Include logging about which buckets we end up nuking
 */
package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

// Structs unmarshalling the raw config file
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

// Structs defining the config object we pass around
type ConfigObj struct {
	S3 rules `yaml:"s3"`
}

type rules struct {
	IncludeRule filterRule `yaml:"include"`
	ExcludeRule filterRule `yaml:"exclude"`
}

type filterRule struct {
	NamesRE []*regexp.Regexp `yaml:"names_regex"`
}

func GetConfig(filePath string) (ConfigObj, error) {
	// TODO: NamesRE might be uninitialized slice
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
