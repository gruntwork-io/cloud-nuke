package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/pterm/pterm"
)

func (csr *ConfigServiceRule) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	configRuleNames := []*string{}

	paginator := func(output *configservice.DescribeConfigRulesOutput, lastPage bool) bool {
		for _, configRule := range output.ConfigRules {
			if configObj.ConfigServiceRule.ShouldInclude(config.ResourceValue{
				Name: configRule.ConfigRuleName,
			}) && *configRule.ConfigRuleState == "ACTIVE" {
				configRuleNames = append(configRuleNames, configRule.ConfigRuleName)
			}
		}

		return !lastPage
	}

	// Pass an empty config rules input, to signify we want all config rules returned
	param := &configservice.DescribeConfigRulesInput{}

	err := csr.Client.DescribeConfigRulesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
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
		_, err := csr.Client.DeleteRemediationConfiguration(&configservice.DeleteRemediationConfigurationInput{
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

		params := &configservice.DeleteConfigRuleInput{
			ConfigRuleName: aws.String(configRuleName),
		}
		_, err = csr.Client.DeleteConfigRule(params)
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
