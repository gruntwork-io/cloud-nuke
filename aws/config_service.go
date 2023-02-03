package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllConfigRules(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]string, error) {
	svc := configservice.New(session)

	configRuleNames := []string{}

	paginator := func(output *configservice.DescribeConfigRulesOutput, lastPage bool) bool {
		for _, configRule := range output.ConfigRules {
			if shouldIncludeConfigRule(configRule, excludeAfter, configObj) {
				configRuleNames = append(configRuleNames, aws.StringValue(configRule.ConfigRuleName))
			}
		}
		return !lastPage
	}

	// Pass an empty config rules input, to signify we want all config rules returned
	param := &configservice.DescribeConfigRulesInput{}

	err := svc.DescribeConfigRulesPages(param, paginator)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return configRuleNames, nil
}

func shouldIncludeConfigRule(configRule *configservice.ConfigRule, excludeAfter time.Time, configObj config.Config) bool {
	if configRule == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(configRule.ConfigRuleName),
		configObj.ConfigServiceRule.IncludeRule.NamesRegExp,
		configObj.ConfigServiceRule.ExcludeRule.NamesRegExp,
	)
}

func nukeAllConfigServiceRules(session *session.Session, configRuleNames []string) error {
	svc := configservice.New(session)

	if len(configRuleNames) == 0 {
		logging.Logger.Debugf("No Config service rules to nuke in region %s", *session.Config.Region)
	}

	var deletedConfigRuleNames []*string

	for _, configRuleName := range configRuleNames {
		params := &configservice.DeleteConfigRuleInput{
			ConfigRuleName: aws.String(configRuleName),
		}
		_, err := svc.DeleteConfigRule(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   configRuleName,
			ResourceType: "Config service rule",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedConfigRuleNames = append(deletedConfigRuleNames, aws.String(configRuleName))
			logging.Logger.Debugf("Deleted Config service rule: %s", configRuleName)
		}
	}

	logging.Logger.Debugf("[OK] %d Config service rules deleted in %s", len(deletedConfigRuleNames), *session.Config.Region)

	return nil
}
