package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ConfigServiceRuleAPI defines the interface for AWS Config Service rule operations.
type ConfigServiceRuleAPI interface {
	DescribeConfigRules(ctx context.Context, params *configservice.DescribeConfigRulesInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigRulesOutput, error)
	DescribeRemediationConfigurations(ctx context.Context, params *configservice.DescribeRemediationConfigurationsInput, optFns ...func(*configservice.Options)) (*configservice.DescribeRemediationConfigurationsOutput, error)
	DeleteRemediationConfiguration(ctx context.Context, params *configservice.DeleteRemediationConfigurationInput, optFns ...func(*configservice.Options)) (*configservice.DeleteRemediationConfigurationOutput, error)
	DeleteConfigRule(ctx context.Context, params *configservice.DeleteConfigRuleInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigRuleOutput, error)
}

// NewConfigServiceRules creates a new Config Service Rules resource using the generic resource pattern.
func NewConfigServiceRules() AwsResource {
	return NewAwsResource(&resource.Resource[ConfigServiceRuleAPI]{
		ResourceTypeName: "config-rules",
		BatchSize:        200,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ConfigServiceRuleAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = configservice.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ConfigServiceRule
		},
		Lister: listConfigServiceRules,
		Nuker:  resource.SequentialDeleter(deleteConfigServiceRule),
	})
}

// listConfigServiceRules retrieves all Config Service rules that match the config filters.
func listConfigServiceRules(ctx context.Context, client ConfigServiceRuleAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var configRuleNames []*string

	paginator := configservice.NewDescribeConfigRulesPaginator(client, &configservice.DescribeConfigRulesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, configRule := range page.ConfigRules {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: configRule.ConfigRuleName,
			}) && configRule.ConfigRuleState == types.ConfigRuleStateActive {
				configRuleNames = append(configRuleNames, configRule.ConfigRuleName)
			}
		}
	}

	return configRuleNames, nil
}

// deleteConfigServiceRule deletes a single Config Service rule.
// It first checks for and deletes any remediation configurations before deleting the rule.
func deleteConfigServiceRule(ctx context.Context, client ConfigServiceRuleAPI, id *string) error {
	ruleName := aws.ToString(id)

	// Step 1: Check for remediation configurations and delete them first
	res, err := client.DescribeRemediationConfigurations(ctx, &configservice.DescribeRemediationConfigurationsInput{
		ConfigRuleNames: []string{ruleName},
	})
	if err != nil {
		return err
	}

	if len(res.RemediationConfigurations) > 0 {
		logging.Debugf("Deleting remediation configuration for config rule: %s", ruleName)
		_, err := client.DeleteRemediationConfiguration(ctx, &configservice.DeleteRemediationConfigurationInput{
			ConfigRuleName: id,
		})
		if err != nil {
			return err
		}
	}

	// Step 2: Delete the config rule
	_, err = client.DeleteConfigRule(ctx, &configservice.DeleteConfigRuleInput{
		ConfigRuleName: id,
	})
	return err
}
