package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
)

func (csr *ConfigServiceRule) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var configRuleNames []*string
	paginator := configservice.NewDescribeConfigRulesPaginator(csr.Client, &configservice.DescribeConfigRulesInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, configRule := range output.ConfigRules {
			if configObj.ConfigServiceRule.ShouldInclude(config.ResourceValue{
				Name: configRule.ConfigRuleName,
			}) && configRule.ConfigRuleState == types.ConfigRuleStateActive {
				configRuleNames = append(configRuleNames, configRule.ConfigRuleName)
			}
		}
	}

	return configRuleNames, nil
}

func (csr *ConfigServiceRule) nukeAll(configRuleNames []string) error {
	if len(configRuleNames) == 0 {
		logging.Debugf("No Config service rules to nuke in region %s", csr.Region)
	}

	var deletedConfigRuleNames []*string

	for _, configRuleName := range configRuleNames {
		logging.Debug(fmt.Sprintf("Start deleting config service rule: %s", configRuleName))

		res, err := csr.Client.DescribeRemediationConfigurations(csr.Context, &configservice.DescribeRemediationConfigurationsInput{
			ConfigRuleNames: []string{configRuleName},
		})
		if err != nil {
			pterm.Error.Println(fmt.Sprintf("Failed to describe remediation configurations w/ err %s", err))
			report.Record(report.Entry{
				Identifier:   configRuleName,
				ResourceType: "Config service rule",
				Error:        err,
			})
			continue
		}

		if len(res.RemediationConfigurations) > 0 {
			_, err := csr.Client.DeleteRemediationConfiguration(csr.Context, &configservice.DeleteRemediationConfigurationInput{
				ConfigRuleName: aws.String(configRuleName),
			})
			if err != nil {
				pterm.Error.Println(fmt.Sprintf("Failed to delete remediation configuration w/ err %s", err))
				report.Record(report.Entry{
					Identifier:   configRuleName,
					ResourceType: "Config service rule",
					Error:        err,
				})

				continue
			}
		}

		params := &configservice.DeleteConfigRuleInput{
			ConfigRuleName: aws.String(configRuleName),
		}
		_, err = csr.Client.DeleteConfigRule(csr.Context, params)
		if err != nil {
			pterm.Error.Println(fmt.Sprintf("Failed to delete config rule w/ err %s", err))
			report.Record(report.Entry{
				Identifier:   configRuleName,
				ResourceType: "Config service rule",
				Error:        err,
			})
		}

		deletedConfigRuleNames = append(deletedConfigRuleNames, aws.String(configRuleName))
		logging.Debug(fmt.Sprintf("Successfully deleted config service rule: %s", configRuleName))
		report.Record(report.Entry{
			Identifier:   configRuleName,
			ResourceType: "Config service rule",
		})
	}

	logging.Debug(
		fmt.Sprintf("Completed deleting %d config service rules %s", len(deletedConfigRuleNames), csr.Region))
	return nil
}
