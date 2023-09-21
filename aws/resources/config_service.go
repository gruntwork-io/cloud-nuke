package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (csr *ConfigServiceRule) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	configRuleNames := []*string{}

	paginator := func(output *configservice.DescribeConfigRulesOutput, lastPage bool) bool {
		for _, configRule := range output.ConfigRules {
			if configObj.ConfigServiceRule.ShouldInclude(config.ResourceValue{
				Name: configRule.ConfigRuleName,
			}) {
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
		logging.Logger.Debugf("No Config service rules to nuke in region %s", csr.Region)
	}

	var deletedConfigRuleNames []*string

	for _, configRuleName := range configRuleNames {
		params := &configservice.DeleteConfigRuleInput{
			ConfigRuleName: aws.String(configRuleName),
		}
		_, err := csr.Client.DeleteConfigRule(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   configRuleName,
			ResourceType: "Config service rule",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Config Service Rule",
			}, map[string]interface{}{
				"region": csr.Region,
			})
		} else {
			deletedConfigRuleNames = append(deletedConfigRuleNames, aws.String(configRuleName))
			logging.Logger.Debugf("Deleted Config service rule: %s", configRuleName)
		}
	}

	logging.Logger.Debugf("[OK] %d Config service rules deleted in %s", len(deletedConfigRuleNames), csr.Region)

	return nil
}
