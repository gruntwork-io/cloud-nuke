package resources

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// ReceiptRuleAllowedRegions lists AWS regions where SES email receiving is supported.
// SES does not support email receiving in many regions. Reference:
// https://docs.aws.amazon.com/ses/latest/dg/regions.html#region-receive-email
var ReceiptRuleAllowedRegions = []string{
	"us-east-1", "us-east-2", "us-west-2", "ap-southeast-3", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	"ca-central-1", "eu-central-1", "eu-west-1", "eu-west-2",
}

// SESReceiptRuleSetAPI defines the interface for SES Receipt Rule Set operations.
type SESReceiptRuleSetAPI interface {
	DescribeActiveReceiptRuleSet(ctx context.Context, params *ses.DescribeActiveReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DescribeActiveReceiptRuleSetOutput, error)
	ListReceiptRuleSets(ctx context.Context, params *ses.ListReceiptRuleSetsInput, optFns ...func(*ses.Options)) (*ses.ListReceiptRuleSetsOutput, error)
	DeleteReceiptRuleSet(ctx context.Context, params *ses.DeleteReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptRuleSetOutput, error)
}

// NewSesReceiptRule creates a new SES Receipt Rule Set resource using the generic resource pattern.
func NewSesReceiptRule() AwsResource {
	return NewAwsResource(&resource.Resource[SESReceiptRuleSetAPI]{
		ResourceTypeName: "ses-receipt-rule-set",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SESReceiptRuleSetAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ses.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SESReceiptRuleSet
		},
		Lister: listSesReceiptRuleSets,
		Nuker:  resource.SimpleBatchDeleter(deleteSesReceiptRuleSet),
	})
}

// listSesReceiptRuleSets retrieves all SES receipt rule sets that match the config filters.
// Note: Active rule sets cannot be deleted, so they are excluded from the results.
func listSesReceiptRuleSets(ctx context.Context, client SESReceiptRuleSetAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Check if region supports email receiving
	if !slices.Contains(ReceiptRuleAllowedRegions, scope.Region) {
		logging.Debugf("Region %s is not allowed for Email receiving", scope.Region)
		return nil, nil
	}

	// Get active rule set to exclude it (active rule sets cannot be deleted)
	activeRule, err := client.DescribeActiveReceiptRuleSet(ctx, &ses.DescribeActiveReceiptRuleSetInput{})
	if err != nil {
		return nil, err
	}

	var ruleSets []*string
	var nextToken *string

	for {
		output, err := client.ListReceiptRuleSets(ctx, &ses.ListReceiptRuleSetsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, ruleSet := range output.RuleSets {
			// Skip active rule set (cannot be deleted)
			if activeRule != nil && activeRule.Metadata != nil &&
				aws.ToString(activeRule.Metadata.Name) == aws.ToString(ruleSet.Name) {
				logging.Debugf("Skipping active ruleset %s (cannot be deleted)", aws.ToString(ruleSet.Name))
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: ruleSet.Name,
				Time: ruleSet.CreatedTimestamp,
			}) {
				ruleSets = append(ruleSets, ruleSet.Name)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return ruleSets, nil
}

// deleteSesReceiptRuleSet deletes a single SES receipt rule set.
func deleteSesReceiptRuleSet(ctx context.Context, client SESReceiptRuleSetAPI, id *string) error {
	_, err := client.DeleteReceiptRuleSet(ctx, &ses.DeleteReceiptRuleSetInput{
		RuleSetName: id,
	})
	return err
}

// SESReceiptFilterAPI defines the interface for SES Receipt Filter operations.
type SESReceiptFilterAPI interface {
	ListReceiptFilters(ctx context.Context, params *ses.ListReceiptFiltersInput, optFns ...func(*ses.Options)) (*ses.ListReceiptFiltersOutput, error)
	DeleteReceiptFilter(ctx context.Context, params *ses.DeleteReceiptFilterInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptFilterOutput, error)
}

// NewSesReceiptFilter creates a new SES Receipt Filter resource using the generic resource pattern.
func NewSesReceiptFilter() AwsResource {
	return NewAwsResource(&resource.Resource[SESReceiptFilterAPI]{
		ResourceTypeName: "ses-receipt-filter",
		BatchSize:        50,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SESReceiptFilterAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ses.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SESReceiptFilter
		},
		Lister: listSesReceiptFilters,
		Nuker:  resource.SimpleBatchDeleter(deleteSesReceiptFilter),
	})
}

// listSesReceiptFilters retrieves all SES receipt filters that match the config filters.
// Note: ListReceiptFilters does not support pagination.
func listSesReceiptFilters(ctx context.Context, client SESReceiptFilterAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.ListReceiptFilters(ctx, &ses.ListReceiptFiltersInput{})
	if err != nil {
		return nil, err
	}

	var filters []*string
	for _, filter := range output.Filters {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: filter.Name,
		}) {
			filters = append(filters, filter.Name)
		}
	}

	return filters, nil
}

// deleteSesReceiptFilter deletes a single SES receipt filter.
func deleteSesReceiptFilter(ctx context.Context, client SESReceiptFilterAPI, id *string) error {
	_, err := client.DeleteReceiptFilter(ctx, &ses.DeleteReceiptFilterInput{
		FilterName: id,
	})
	return err
}
