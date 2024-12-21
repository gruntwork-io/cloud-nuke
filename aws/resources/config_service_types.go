package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ConfigServiceRuleAPI interface {
	DescribeConfigRules(ctx context.Context, params *configservice.DescribeConfigRulesInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigRulesOutput, error)
	DescribeRemediationConfigurations(ctx context.Context, params *configservice.DescribeRemediationConfigurationsInput, optFns ...func(*configservice.Options)) (*configservice.DescribeRemediationConfigurationsOutput, error)
	DeleteRemediationConfiguration(ctx context.Context, params *configservice.DeleteRemediationConfigurationInput, optFns ...func(*configservice.Options)) (*configservice.DeleteRemediationConfigurationOutput, error)
	DeleteConfigRule(ctx context.Context, params *configservice.DeleteConfigRuleInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigRuleOutput, error)
}

type ConfigServiceRule struct {
	BaseAwsResource
	Client    ConfigServiceRuleAPI
	Region    string
	RuleNames []string
}

func (csr *ConfigServiceRule) InitV2(cfg aws.Config) {
	csr.Client = configservice.NewFromConfig(cfg)
}

func (csr *ConfigServiceRule) ResourceName() string {
	return "config-rules"
}

func (csr *ConfigServiceRule) ResourceIdentifiers() []string {
	return csr.RuleNames
}

func (csr *ConfigServiceRule) MaxBatchSize() int {
	return 200
}

func (csr *ConfigServiceRule) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ConfigServiceRule
}

func (csr *ConfigServiceRule) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := csr.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	csr.RuleNames = aws.ToStringSlice(identifiers)
	return csr.RuleNames, nil
}

func (csr *ConfigServiceRule) Nuke(identifiers []string) error {
	if err := csr.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
