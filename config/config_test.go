package config

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
		ResourceType{FilterRule{}, FilterRule{}},
		KMSCustomerKeyResourceType{false, ResourceType{FilterRule{}, FilterRule{}}},
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

// ELBv2Tests

func TestConfigELBv2_Empty(t *testing.T) {
	configFilePath := "./mocks/elbv2_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.ELBv2)
	}

	return
}

func TestConfigELBv2_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/elbv2_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigELBv2_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/elbv2_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigELBv2_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/elbv2_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ELBv2.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ELBv2 regexes, %+v\n", configObj)
	}

	return
}

func TestConfigELBv2_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/elbv2_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ELBv2.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ELBv2 regexes, %+v\n", configObj)
	}

	return
}

func TestConfigELBv2_FilterNames(t *testing.T) {
	configFilePath := "./mocks/elbv2_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ELBv2.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.ELBv2.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ELBv2 regexes, %+v\n", configObj)
	}

	return
}

// SageMakerNotebookTests

func TestConfigSageMakerNotebook_Empty(t *testing.T) {
	configFilePath := "./mocks/sagemakernotebook_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.SageMakerNotebook)
	}

	return
}

func TestConfigSageMakerNotebook_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/sagemakernotebook_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigSageMakerNotebook_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/sagemakernotebook_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigSageMakerNotebook_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/sagemakernotebook_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SageMakerNotebook.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain SageMakerNotebook regexes, %+v\n", configObj)
	}

	return
}

func TestConfigSageMakerNotebook_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/sagemakernotebook_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SageMakerNotebook.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain SageMakerNotebook regexes, %+v\n", configObj)
	}

	return
}

func TestConfigSageMakerNotebook_FilterNames(t *testing.T) {
	configFilePath := "./mocks/sagemakernotebook_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.SageMakerNotebook.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.SageMakerNotebook.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain SageMakerNotebook regexes, %+v\n", configObj)
	}

	return
}

// APIGateway Tests

func TestConfigAPIGateway_Empty(t *testing.T) {
	configFilePath := "./mocks/apigateway_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.APIGateway)
	}

	return
}

func TestConfigAPIGateway_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/apigateway_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGateway_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/apigateway_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGateway_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/apigateway_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.APIGateway.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain APIGateway regexes, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGateway_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/apigateway_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.APIGateway.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain APIGateway regexes, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGateway_FilterNames(t *testing.T) {
	configFilePath := "./mocks/apigateway_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.APIGateway.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.APIGateway.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain APIGateway regexes, %+v\n", configObj)
	}

	return
}

// end APIGateway tests

// APIGateway V2 tests

func TestConfigAPIGatewayV2_Empty(t *testing.T) {
	configFilePath := "./mocks/apigatewayv2_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.APIGateway)
	}

	return
}

func TestConfigAPIGatewayV2_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/apigatewayv2_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGatewayV2_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/apigatewayv2_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGatewayV2_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/apigatewayv2_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.APIGatewayV2.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain APIGatewayV2 regexes, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGatewayV2_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/apigatewayv2_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.APIGatewayV2.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain APIGatewayV2 regexes, %+v\n", configObj)
	}

	return
}

func TestConfigAPIGatewayV2_FilterNames(t *testing.T) {
	configFilePath := "./mocks/apigatewayv2_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.APIGatewayV2.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.APIGatewayV2.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain APIGatewayV2 regexes, %+v\n", configObj)
	}

	return
}

// Elastic FileSystem Tests

func TestConfigElasticFileSystem_Empty(t *testing.T) {
	configFilePath := "./mocks/efs_empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj.APIGateway)
	}

	return
}

func TestConfigElasticFileSystem_EmptyFilters(t *testing.T) {
	configFilePath := "./mocks/efs_empty_filters.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigElasticFileSystem_EmptyRules(t *testing.T) {
	configFilePath := "./mocks/efs_empty_rules.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfigElasticFileSystem_IncludeNames(t *testing.T) {
	configFilePath := "./mocks/efs_include_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ElasticFileSystem.IncludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ElasticFileSystem regexes, %+v\n", configObj)
	}

	return
}

func TestConfigElasticFileSystem_ExcludeNames(t *testing.T) {
	configFilePath := "./mocks/efs_exclude_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ElasticFileSystem.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ElasticFileSystem regexes, %+v\n", configObj)
	}

	return
}

func TestConfigElasticFileSystem_FilterNames(t *testing.T) {
	configFilePath := "./mocks/efs_filter_names.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should not be empty, %+v\n", configObj)
	}

	if len(configObj.ElasticFileSystem.IncludeRule.NamesRegExp) == 0 ||
		len(configObj.ElasticFileSystem.ExcludeRule.NamesRegExp) == 0 {
		assert.Fail(t, "ConfigObj should contain ElasticFileSystem regexes, %+v\n", configObj)
	}

	return
}

// end ElasticFileSystem tests

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

func TestShouldIncludeBasedOnTime_IncludeTimeBefore(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		IncludeRule: FilterRule{TimeBefore: &now},
	}
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_IncludeTimeAfter(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		IncludeRule: FilterRule{TimeAfter: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_ExcludeTimeBefore(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		ExcludeRule: FilterRule{TimeBefore: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_ExcludeTimeAfter(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		ExcludeRule: FilterRule{TimeAfter: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
}

func TestShouldInclude_NameAndTimeFilter(t *testing.T) {
	now := time.Now()

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}
	r := ResourceType{
		ExcludeRule: FilterRule{
			NamesRegExp: excludeREs,
			TimeAfter:   &now,
		},
	}

	// Filter by Time
	assert.False(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("hello_world"),
		Time: aws.Time(now.Add(1)),
	}))
	// Filter by Name
	assert.False(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("test_hello_world"),
		Time: aws.Time(now.Add(1)),
	}))
	// Pass filters
	assert.True(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("hello_world"),
		Time: aws.Time(now.Add(-1)),
	}))
}

func TestAddIncludeAndExcludeAfterTime(t *testing.T) {
	now := aws.Time(time.Now())

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	testConfig := &Config{}
	testConfig.ACM = ResourceType{
		ExcludeRule: FilterRule{
			NamesRegExp: excludeREs,
			TimeAfter:   now,
		},
	}

	testConfig.AddExcludeAfterTime(now)
	assert.Equal(t, testConfig.ACM.ExcludeRule.NamesRegExp, excludeREs)
	assert.Equal(t, testConfig.ACM.ExcludeRule.TimeAfter, now)
	assert.Nil(t, testConfig.ACM.IncludeRule.TimeAfter)

	testConfig.AddIncludeAfterTime(now)
	assert.Equal(t, testConfig.ACM.ExcludeRule.NamesRegExp, excludeREs)
	assert.Equal(t, testConfig.ACM.ExcludeRule.TimeAfter, now)
	assert.Equal(t, testConfig.ACM.IncludeRule.TimeAfter, now)
}
