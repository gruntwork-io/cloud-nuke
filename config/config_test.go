package config

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
	}
}

func TestConfig_Garbage(t *testing.T) {
	configFilePath := "./mocks/garbage.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfig_Malformed(t *testing.T) {
	configFilePath := "./mocks/malformed.yaml"
	_, err := GetConfig(configFilePath)

	// Expect malformed to throw a yaml TypeError
	require.Error(t, err, "Received expected error")
	return
}

func TestConfig_Empty(t *testing.T) {
	configFilePath := "./mocks/empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

// S3 Tests

func TestConfigS3_Empty(t *testing.T) {
	configFilePath := "./mocks/s3_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.S3)
	}

	return
}

func TestConfigS3_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/s3_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/s3_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigS3_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigS3_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/s3_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigS3_FilterNames(t *testing.T) {
	configFilePath := "./mocks/s3_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.S3.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.S3.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain S3 names regexes, %+v\n", configObj)
	}

	return
}

// IAM Users Tests

func TestConfigIAM_Users_Empty(t *testing.T) {
	configFilePath := "./mocks/iam_users_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.IAMUsers)
	}

	return
}

func TestConfigIAM_Users_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/iam_users_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/iam_users_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/iam_users_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMUsers.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/iam_users_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMUsers.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM names regexes, %+v\n", configObj)
	}

	return
}

func TestConfigIAM_Users_FilterNames(t *testing.T) {
	configFilePath := "./mocks/iam_users_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.IAMUsers.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.IAMUsers.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain IAM names regexes, %+v\n", configObj)
	}

	return
}

// Secrets Manager Tests

func TestConfigSecretsManager_Empty(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.SecretsManagerSecrets)
	}

	return
}

func TestConfigSecretsManager_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SecretsManagerSecrets.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain secrets regexes, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SecretsManagerSecrets.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain secrets regexes, %+v\n", configObj)
	}

	return
}

func TestConfigSecretsManager_FilterNames(t *testing.T) {
	configFilePath := "./mocks/secrets_manager_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SecretsManagerSecrets.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.SecretsManagerSecrets.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain secrets regexes, %+v\n", configObj)
	}

	return
}

// DynamoDB Tests

func TestConfigDynamoDB_Empty(t *testing.T) {
	configFilePath := "./mocks/dynamodb_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.DynamoDB)
	}

	return
}

func TestConfigDynamoDB_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/dynamodb_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigDynamoDB_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/dynamodb_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigDynamoDB_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/dynamodb_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.DynamoDB.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain DynamoDB table name regexes, %+v\n", configObj)
	}

	return
}

func TestConfigDynamoDB_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/dynamodb_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.DynamoDB.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain DynamoDB table name regexes, %+v\n", configObj)
	}

	return
}

func TestConfigDynamoDB_FilterNames(t *testing.T) {
	configFilePath := "./mocks/dynamodb_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.DynamoDB.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.DynamoDB.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain DynamoDB table name regexes, %+v\n", configObj)
	}

	return
}

func TestShouldInclude_AllowWhenEmpty(t *testing.T) {
	var includeREs []Expression
	var excludeREs []Expression

	assert.True(t, ShouldInclude("test-open-vpn", includeREs, excludeREs),
		"Should include when both lists are empty")
}

func TestShouldInclude_ExcludeWhenMatches(t *testing.T) {
	var includeREs []Expression

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.False(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should not include when matches from the 'exclude' list")
	assert.True(t, ShouldInclude("tf-state-bucket", includeREs, excludeREs),
		"Should include when doesn't matches from the 'exclude' list")
}

func TestShouldInclude_IncludeWhenMatches(t *testing.T) {
	include, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	var excludeREs []Expression

	assert.True(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should include when matches the 'include' list")
	assert.False(t, ShouldInclude("test-vpc-123", includeREs, excludeREs),
		"Should not include when doesn't matches the 'include' list")
}

func TestShouldInclude_WhenMatchesIncludeAndExclude(t *testing.T) {
	include, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	exclude, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.True(t, ShouldInclude("test-eks-cluster-123", includeREs, excludeREs),
		"Should include when matches the 'include' list but not matches the 'exclude' list")
	assert.False(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should not include when matches 'exclude' list")
	assert.False(t, ShouldInclude("terraform-tf-state", includeREs, excludeREs),
		"Should not include when doesn't matches 'include' list")
}
