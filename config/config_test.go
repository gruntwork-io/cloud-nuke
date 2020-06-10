package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) > 0 {
		parsedLogLevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logging.Logger.Errorf("Invalid log level - %s - %s", logLevel, err)
			os.Exit(1)
		}
		logging.Logger.Level = parsedLogLevel
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func emptyConfigObj() ConfigObj {
	return ConfigObj{Rules{FilterRule{}, FilterRule{}}}
}

func TestConfig_Empty(t *testing.T) {
	configFilePath := "./mocks/empty.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if !reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_Empty(t *testing.T) {
	configFilePath := "./mocks/s3_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if !reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should be empty, %+v\n", configObj.S3)
	}

	return
}

func TestConfigS3_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/s3_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if !reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/s3_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if !reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.IncludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigS3_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigS3_FilterNames(t *testing.T) {
	configFilePath := "./mocks/s3_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	if err != nil {
		assert.Failf(t, "Error reading config - %s - %s", configFilePath, err)
	}

	if reflect.DeepEqual(configObj, emptyConfigObj()) {
		assert.Fail(t, "ConfigObj should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.IncludeRule.NamesRE) == 0 ||
		len(configObj.S3.ExcludeRule.NamesRE) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}
