package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockConfigServiceRuleClient struct {
	DescribeConfigRulesOutput               configservice.DescribeConfigRulesOutput
	DescribeRemediationConfigurationsOutput configservice.DescribeRemediationConfigurationsOutput
	DeleteRemediationConfigurationOutput    configservice.DeleteRemediationConfigurationOutput
	DeleteConfigRuleOutput                  configservice.DeleteConfigRuleOutput
}

func (m *mockConfigServiceRuleClient) DescribeConfigRules(ctx context.Context, params *configservice.DescribeConfigRulesInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigRulesOutput, error) {
	return &m.DescribeConfigRulesOutput, nil
}

func (m *mockConfigServiceRuleClient) DescribeRemediationConfigurations(ctx context.Context, params *configservice.DescribeRemediationConfigurationsInput, optFns ...func(*configservice.Options)) (*configservice.DescribeRemediationConfigurationsOutput, error) {
	return &m.DescribeRemediationConfigurationsOutput, nil
}

func (m *mockConfigServiceRuleClient) DeleteRemediationConfiguration(ctx context.Context, params *configservice.DeleteRemediationConfigurationInput, optFns ...func(*configservice.Options)) (*configservice.DeleteRemediationConfigurationOutput, error) {
	return &m.DeleteRemediationConfigurationOutput, nil
}

func (m *mockConfigServiceRuleClient) DeleteConfigRule(ctx context.Context, params *configservice.DeleteConfigRuleInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigRuleOutput, error) {
	return &m.DeleteConfigRuleOutput, nil
}

func TestListConfigServiceRules(t *testing.T) {
	t.Parallel()

	testName1 := "test-rule-1"
	testName2 := "test-rule-2"

	mock := &mockConfigServiceRuleClient{
		DescribeConfigRulesOutput: configservice.DescribeConfigRulesOutput{
			ConfigRules: []types.ConfigRule{
				{ConfigRuleName: aws.String(testName1), ConfigRuleState: types.ConfigRuleStateActive},
				{ConfigRuleName: aws.String(testName2), ConfigRuleState: types.ConfigRuleStateActive},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listConfigServiceRules(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteConfigServiceRule(t *testing.T) {
	t.Parallel()

	mock := &mockConfigServiceRuleClient{
		DescribeRemediationConfigurationsOutput: configservice.DescribeRemediationConfigurationsOutput{},
		DeleteConfigRuleOutput:                  configservice.DeleteConfigRuleOutput{},
	}

	err := deleteConfigServiceRule(context.Background(), mock, aws.String("test-rule"))
	require.NoError(t, err)
}

func TestDeleteConfigServiceRule_WithRemediation(t *testing.T) {
	t.Parallel()

	mock := &mockConfigServiceRuleClient{
		DescribeRemediationConfigurationsOutput: configservice.DescribeRemediationConfigurationsOutput{
			RemediationConfigurations: []types.RemediationConfiguration{
				{ConfigRuleName: aws.String("test-rule")},
			},
		},
		DeleteRemediationConfigurationOutput: configservice.DeleteRemediationConfigurationOutput{},
		DeleteConfigRuleOutput:               configservice.DeleteConfigRuleOutput{},
	}

	err := deleteConfigServiceRule(context.Background(), mock, aws.String("test-rule"))
	require.NoError(t, err)
}
