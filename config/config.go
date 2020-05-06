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

type RawConfig struct {
  ResourceType resourceType `yaml:"s3"`
}

type resourceType struct {
  MatchingRule []string `yaml:"include_names_regex"`
}

type ConfigObj struct {
  S3FilterRule filterRule `yaml:"s3"`
}

type filterRule struct {
  IncludeNamesRE []*regexp.Regexp `yaml:"include_names_regex"`
}

func GetConfig(filePath string) (ConfigObj, error) {
  // TODO: IncludeNamesRE might be uninitialized slice
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

  for _, pattern := range rawConfig.ResourceType.MatchingRule {
    re, err := regexp.Compile(pattern)
    if err != nil {
      return ConfigObj{}, err
    }

    configObj.S3FilterRule.IncludeNamesRE = append(configObj.S3FilterRule.IncludeNamesRE, re)
  }

	return configObj, nil
}
