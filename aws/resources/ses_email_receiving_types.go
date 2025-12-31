package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type SESEmailReceivingAPI interface {
	DescribeActiveReceiptRuleSet(ctx context.Context, params *ses.DescribeActiveReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DescribeActiveReceiptRuleSetOutput, error)
	ListReceiptRuleSets(ctx context.Context, params *ses.ListReceiptRuleSetsInput, optFns ...func(*ses.Options)) (*ses.ListReceiptRuleSetsOutput, error)
	DeleteReceiptRuleSet(ctx context.Context, params *ses.DeleteReceiptRuleSetInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptRuleSetOutput, error)
	ListReceiptFilters(ctx context.Context, params *ses.ListReceiptFiltersInput, optFns ...func(*ses.Options)) (*ses.ListReceiptFiltersOutput, error)
	DeleteReceiptFilter(ctx context.Context, params *ses.DeleteReceiptFilterInput, optFns ...func(*ses.Options)) (*ses.DeleteReceiptFilterOutput, error)
}

type SesReceiptFilter struct {
	BaseAwsResource
	Client  SESEmailReceivingAPI
	Region  string
	Ids     []string
	Nukable map[string]bool
}

func (sef *SesReceiptFilter) Init(cfg aws.Config) {
	sef.Client = ses.NewFromConfig(cfg)
	sef.Nukable = map[string]bool{}
}

// ResourceName - the simple name of the aws resource
func (sef *SesReceiptFilter) ResourceName() string {
	return "ses-receipt-filter"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (sef *SesReceiptFilter) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The Ids of the receipt filter
func (sef *SesReceiptFilter) ResourceIdentifiers() []string {
	return sef.Ids
}

func (sef *SesReceiptFilter) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESReceiptFilter
}

func (sef *SesReceiptFilter) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sef.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sef.Ids = aws.ToStringSlice(identifiers)
	return sef.Ids, nil
}

// Nuke - nuke 'em all!!!
func (sef *SesReceiptFilter) Nuke(ctx context.Context, identifiers []string) error {
	if err := sef.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// SesReceiptRule - represents all ses receipt rules
type SesReceiptRule struct {
	BaseAwsResource
	Client  SESEmailReceivingAPI
	Region  string
	Ids     []string
	Nukable map[string]bool
}

func (sef *SesReceiptRule) Init(cfg aws.Config) {
	sef.Client = ses.NewFromConfig(cfg)
	sef.Nukable = map[string]bool{}
}

// ResourceName - the simple name of the aws resource
func (ser *SesReceiptRule) ResourceName() string {
	return "ses-receipt-rule-set"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (ser *SesReceiptRule) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The names of the rule set
func (ser *SesReceiptRule) ResourceIdentifiers() []string {
	return ser.Ids
}

func (sef *SesReceiptRule) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.SESReceiptRuleSet
}

func (ser *SesReceiptRule) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ser.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ser.Ids = aws.ToStringSlice(identifiers)
	return ser.Ids, nil
}

// Nuke - nuke 'em all!!!
func (ser *SesReceiptRule) Nuke(ctx context.Context, identifiers []string) error {
	if err := ser.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
