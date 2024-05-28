package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedConfigServiceRule struct {
	configserviceiface.ConfigServiceAPI
	DescribeConfigRulesOutput            configservice.DescribeConfigRulesOutput
	DeleteConfigRuleOutput               configservice.DeleteConfigRuleOutput
	DeleteRemediationConfigurationOutput configservice.DeleteRemediationConfigurationOutput
}

func (m mockedConfigServiceRule) DescribeConfigRulesPagesWithContext(_ awsgo.Context, _ *configservice.DescribeConfigRulesInput, fn func(*configservice.DescribeConfigRulesOutput, bool) bool, _ ...request.Option) error {
	fn(&m.DescribeConfigRulesOutput, true)
	return nil
}

func (m mockedConfigServiceRule) DeleteConfigRuleWithContext(_ awsgo.Context, _ *configservice.DeleteConfigRuleInput, _ ...request.Option) (*configservice.DeleteConfigRuleOutput, error) {
	return &m.DeleteConfigRuleOutput, nil
}

func (m mockedConfigServiceRule) DeleteRemediationConfigurationWithContext(_ awsgo.Context, _ *configservice.DeleteRemediationConfigurationInput, _ ...request.Option) (*configservice.DeleteRemediationConfigurationOutput, error) {
	return &m.DeleteRemediationConfigurationOutput, nil
}

func TestConfigServiceRule_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-rule-1"
	testName2 := "test-rule-2"
	csr := ConfigServiceRule{
		Client: mockedConfigServiceRule{
			DescribeConfigRulesOutput: configservice.DescribeConfigRulesOutput{
				ConfigRules: []*configservice.ConfigRule{
					{ConfigRuleName: aws.String(testName1), ConfigRuleState: aws.String("ACTIVE")},
					{ConfigRuleName: aws.String(testName2), ConfigRuleState: aws.String("ACTIVE")},
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestConfigServiceRule_NukeAll(t *testing.T) {

	t.Parallel()

	csr := ConfigServiceRule{
		Client: mockedConfigServiceRule{
			DeleteConfigRuleOutput:               configservice.DeleteConfigRuleOutput{},
			DeleteRemediationConfigurationOutput: configservice.DeleteRemediationConfigurationOutput{},
		},
	}

	err := csr.nukeAll([]string{"test"})
	require.NoError(t, err)
}
