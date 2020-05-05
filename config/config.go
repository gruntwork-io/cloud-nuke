/* check file existence (handle file doesn't exist)
* parse yaml (standard go yaml parser)
* validate that yaml conforms to the spec
* Include logging about which buckets we end up nuking
 */
package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
)

type ConfigObj struct {
  ResourceType resourceType `yaml:"s3"`
}

type resourceType struct {
  MatchingRule []string `yaml:"include_names_regex"`
}

func GetConfig() ConfigObj {
	absolutePath, err := filepath.Abs("config/mocks/s3_include_names.yaml")
	if err != nil {
		log.Printf("filepath.Abs err   #%v ", err)
	}

	yamlFile, err := ioutil.ReadFile(absolutePath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	config := ConfigObj{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- unmarshalled:\n%v\n\n", config)

	marshalled, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- marshalled:\n%s\n\n", string(marshalled))

	return config

	//   m := make(map[interface{}]interface{})
	//
	//   err = yaml.Unmarshal([]byte(data), &m)
	//   if err != nil {
	//     log.Fatalf("error: %v", err)
	//   }
	//   fmt.Printf("--- m:\n%v\n\n", m)
	//
	//   d, err = yaml.Marshal(&m)
	//   if err != nil {
	//     log.Fatalf("error: %v", err)
	//   }
	//   fmt.Printf("--- m dump:\n%s\n\n", string(d))
}
