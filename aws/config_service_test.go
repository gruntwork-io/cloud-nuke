package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListConfigServiceRules(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configRuleName := createConfigServiceRule(t, region)
	defer deleteConfigServiceRule(t, region, configRuleName, false)

	configServiceRuleNames, err := getAllConfigRules(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, configServiceRuleNames, configRuleName)
}

func TestNukeConfigServiceRuleOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configRuleName := createConfigServiceRule(t, region)
	defer deleteConfigServiceRule(t, region, configRuleName, false)

	configServiceRuleNames := []string{configRuleName}

	require.NoError(
		t,
		nukeAllConfigServiceRules(session, configServiceRuleNames),
	)

	assertConfigServiceRulesDeleted(t, region, configServiceRuleNames)
}

func TestNukeConfigServiceRuleMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configServiceRuleNames := []string{}
	for i := 0; i < 3; i++ {
		configServiceRuleName := createConfigServiceRule(t, region)
		defer deleteConfigServiceRule(t, region, configServiceRuleName, false)
		configServiceRuleNames = append(configServiceRuleNames, configServiceRuleName)
	}

	require.NoError(
		t,
		nukeAllConfigServiceRules(session, configServiceRuleNames),
	)

	assertConfigServiceRulesDeleted(t, region, configServiceRuleNames)
}

// Test helpers

// ensureConfigurationRecorderExistsInRegion is a convenience method to be used during testing
// since you cannot create a custom configuration rule unless you already have a configuration
// recorder in the target region
func ensureConfigurationRecorderExistsInRegion(t *testing.T, region string) string {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configService := configservice.New(session)

	param := &configservice.DescribeConfigurationRecordersInput{}

	output, lookupErr := configService.DescribeConfigurationRecorders(param)
	require.NoError(t, lookupErr)

	// If we have no recorders in the region, create one - otherwise we have no more work to do
	if len(output.ConfigurationRecorders) == 0 {
		configRecorderName := createConfigurationRecorder(t, region)
		require.NotEqual(t, "", configRecorderName)
		return configRecorderName
	}

	return aws.StringValue(output.ConfigurationRecorders[0].Name)
}

// getAccountId contacts STS to retrieve the ID of the AWS account the tests are
// being run against. This is necessary because we need to "fake" an IAM role,
// but the fake IAM role must contain the correct account ID to avoid a cross-account
// validation error
func getAccountId(t *testing.T, session *session.Session) string {
	stsService := sts.New(session)
	stsOutput, stsErr := stsService.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	require.NoError(t, stsErr)

	return aws.StringValue(stsOutput.Account)
}

// scaffoldTestConfigRecorder is a convenience method that generates a custom config service rule
// suitable for use in testing
func scaffoldTestConfigRecorder(t *testing.T, session *session.Session) *configservice.ConfigurationRecorder {
	testConfigurationRecorderName := "default"

	accountId := getAccountId(t, session)
	// Create the test Configuration recorder
	testConfigurationRecorder := &configservice.ConfigurationRecorder{
		Name: aws.String(testConfigurationRecorderName),
		RecordingGroup: &configservice.RecordingGroup{
			AllSupported:               aws.Bool(false),
			IncludeGlobalResourceTypes: aws.Bool(false),
			ResourceTypes: []*string{
				aws.String("AWS::Lambda::Function"),
			},
		},
		// As the AWS documentation for config service notes, this RoleARN is not required by
		// the API model, but it IS required by the AWS API, meaning you must pass a value
		// We can "fake" this IAM role, in the sense that it doesn't actually need to exist,
		// However, the AWS API will return an error if you don't supply the correct account
		// number in the role ARN. Therefore, we need to look up the current account's ID dynamically
		RoleARN: aws.String(fmt.Sprintf("arn:aws:iam::%s:role/S3Access", accountId)),
	}
	return testConfigurationRecorder
}

func createConfigurationRecorder(t *testing.T, region string) string {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configService := configservice.New(session)

	testConfigRecorder := scaffoldTestConfigRecorder(t, session)

	param := &configservice.PutConfigurationRecorderInput{
		ConfigurationRecorder: testConfigRecorder,
	}

	// The PutConfigurationRecorderOutput is an empty struct
	_, putErr := configService.PutConfigurationRecorder(param)
	require.NoError(t, putErr)

	return aws.StringValue(testConfigRecorder.Name)
}

// scaffoldConfigServiceRule is a convenience method that generates an essentially nonsensical custom config rule
// It is intended for use during testing
func scaffoldConfigServiceRule() *configservice.ConfigRule {
	configServiceRuleName := strings.ToLower(fmt.Sprintf("cloud-nuke-test-%s-%s", util.UniqueID(), util.UniqueID()))

	configRulePolicyText := `
# This rule checks if point in time recovery (PITR) is enabled on active Amazon DynamoDB tables
let status = ['ACTIVE']

rule tableisactive when
    resourceType == "AWS::DynamoDB::Table" {
    configuration.tableStatus == %status
}

rule checkcompliance when
    resourceType == "AWS::DynamoDB::Table"
    tableisactive {
        let pitr = supplementaryConfiguration.ContinuousBackupsDescription.pointInTimeRecoveryDescription.pointInTimeRecoveryStatus
        %pitr == "ENABLED"
}
	`
	sourceDetail := &configservice.SourceDetail{
		EventSource: aws.String("aws.config"),
		MessageType: aws.String("ConfigurationItemChangeNotification"),
	}

	configRuleSource := &configservice.Source{
		CustomPolicyDetails: &configservice.CustomPolicyDetails{
			PolicyRuntime: aws.String("guard-2.x.x"),
			PolicyText:    aws.String(configRulePolicyText),
		},
		Owner: aws.String("CUSTOM_POLICY"),
		SourceDetails: []*configservice.SourceDetail{
			sourceDetail,
		},
		SourceIdentifier: aws.String("cloud-nuke-test"),
	}

	return &configservice.ConfigRule{
		ConfigRuleName: aws.String(configServiceRuleName),
		Source:         configRuleSource,
	}
}

// createConfigServiceRule creates a custom AWS config service rule
func createConfigServiceRule(t *testing.T, region string) string {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	// We must first ensure there is a configuration recorder in the target region
	// Otherwise, the attempt to create the custom configuration rule will fail
	ensureConfigurationRecorderExistsInRegion(t, region)

	configService := configservice.New(session)

	testConfigServiceRule := scaffoldConfigServiceRule()

	param := &configservice.PutConfigRuleInput{
		ConfigRule: testConfigServiceRule,
	}

	// PutConfigRuleOutput is an empty struct, so we just check for an error
	_, createConfigServiceRuleErr := configService.PutConfigRule(param)
	require.NoError(t, createConfigServiceRuleErr)

	return *testConfigServiceRule.ConfigRuleName
}

func deleteConfigServiceRule(t *testing.T, region string, configServiceRuleName string, checkErr bool) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	configService := configservice.New(session)

	param := &configservice.DeleteConfigRuleInput{
		ConfigRuleName: aws.String(configServiceRuleName),
	}

	_, deleteErr := configService.DeleteConfigRule(param)
	if checkErr {
		require.NoError(t, deleteErr)
	}
}

func assertConfigServiceRulesDeleted(t *testing.T, region string, configServiceRuleNames []string) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	svc := configservice.New(session)

	param := &configservice.DescribeConfigRulesInput{
		ConfigRuleNames: aws.StringSlice(configServiceRuleNames),
	}

	resp, err := svc.DescribeConfigRules(param)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case configservice.ErrCodeNoSuchConfigRuleException:
				t.Log("Ignoring Config service rule not found error in test lookup")
			default:
				require.NoError(t, err)
				if len(resp.ConfigRules) > 0 {
					configServiceRuleNames := []string{}
					// Unpack config service rules that failed to delete
					for _, configRule := range resp.ConfigRules {
						configServiceRuleNames = append(configServiceRuleNames, aws.StringValue(configRule.ConfigRuleName))
					}
					t.Fatalf("At least one of the following Config service rules was not deleted: %+v\n", aws.StringSlice(configServiceRuleNames))
				}
			}
		}
	}
}
