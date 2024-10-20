package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedConfigServiceRule struct {
	ConfigServiceRuleAPI
	DescribeConfigRulesOutput               configservice.DescribeConfigRulesOutput
	DescribeRemediationConfigurationsOutput configservice.DescribeRemediationConfigurationsOutput
	DeleteRemediationConfigurationOutput    configservice.DeleteRemediationConfigurationOutput
	DeleteConfigRuleOutput                  configservice.DeleteConfigRuleOutput
}

func (m mockedConfigServiceRule) DescribeConfigRules(ctx context.Context, params *configservice.DescribeConfigRulesInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigRulesOutput, error) {
	return &m.DescribeConfigRulesOutput, nil
}

func (m mockedConfigServiceRule) DescribeRemediationConfigurations(ctx context.Context, params *configservice.DescribeRemediationConfigurationsInput, optFns ...func(*configservice.Options)) (*configservice.DescribeRemediationConfigurationsOutput, error) {
	return &m.DescribeRemediationConfigurationsOutput, nil
}

func (m mockedConfigServiceRule) DeleteRemediationConfiguration(ctx context.Context, params *configservice.DeleteRemediationConfigurationInput, optFns ...func(*configservice.Options)) (*configservice.DeleteRemediationConfigurationOutput, error) {
	return &m.DeleteRemediationConfigurationOutput, nil
}

func (m mockedConfigServiceRule) DeleteConfigRule(ctx context.Context, params *configservice.DeleteConfigRuleInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigRuleOutput, error) {
	return &m.DeleteConfigRuleOutput, nil
}

func TestConfigServiceRule_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-rule-1"
	testName2 := "test-rule-2"
	csr := ConfigServiceRule{
		Client: mockedConfigServiceRule{
			DescribeConfigRulesOutput: configservice.DescribeConfigRulesOutput{
				ConfigRules: []types.ConfigRule{
					{ConfigRuleName: aws.String(testName1), ConfigRuleState: types.ConfigRuleStateActive},
					{ConfigRuleName: aws.String(testName2), ConfigRuleState: types.ConfigRuleStateActive},
				},
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
					}}},
			},
			expected: []string{testName2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := csr.getAll(context.Background(), config.Config{
				ConfigServiceRule: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestConfigServiceRule_NukeAll(t *testing.T) {
	t.Parallel()

	csr := ConfigServiceRule{
		Client: mockedConfigServiceRule{
			DeleteConfigRuleOutput:                  configservice.DeleteConfigRuleOutput{},
			DeleteRemediationConfigurationOutput:    configservice.DeleteRemediationConfigurationOutput{},
			DescribeRemediationConfigurationsOutput: configservice.DescribeRemediationConfigurationsOutput{},
		},
	}

	err := csr.nukeAll([]string{"test"})
	require.NoError(t, err)
}
