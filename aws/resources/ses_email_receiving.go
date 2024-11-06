package resources

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

var ReceiptRuleAllowedRegions = []string{
	"us-east-1", "us-east-2", "us-west-2", "ap-southeast-3", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	"ca-central-1", "eu-central-1", "eu-west-1", "eu-west-2",
}

// Returns a formatted string of receipt rule names
func (s *SesReceiptRule) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	// NOTE : SES does not support email receiving in the following Regions: US West (N. California), Africa (Cape Town), Asia Pacific (Mumbai), Asia Pacific (Osaka),
	// Asia Pacific (Seoul), Europe (Milan), Europe (Paris), Europe (Stockholm), Israel (Tel Aviv), Middle East (Bahrain), South America (São Paulo),
	// AWS GovCloud (US-West), and AWS GovCloud (US-East).
	// Reference : https://docs.aws.amazon.com/ses/latest/dg/regions.html#region-receive-email

	// since the aws doesn't provide an api to check the ses receipt rule is allowed for the region
	if !slices.Contains(ReceiptRuleAllowedRegions, s.Region) {
		logging.Debugf("Region %s is not allowed for Email receiving", s.Region)
		return nil, nil
	}

	// https://docs.aws.amazon.com/cli/latest/reference/ses/delete-receipt-rule-set.html
	// Important : The currently active rule set cannot be deleted.
	activeRule, err := s.Client.DescribeActiveReceiptRuleSet(s.Context, &ses.DescribeActiveReceiptRuleSetInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	result, err := s.Client.ListReceiptRuleSets(s.Context, &ses.ListReceiptRuleSetsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var rulesets []*string
	for _, sets := range result.RuleSets {
		// checking the rule set is the active one
		if activeRule != nil && activeRule.Metadata != nil && aws.ToString(activeRule.Metadata.Name) == aws.ToString(sets.Name) {
			logging.Debugf("The Ruleset %s is active and you wont able to delete it", aws.ToString(sets.Name))
			continue
		}
		if configObj.SESReceiptRuleSet.ShouldInclude(config.ResourceValue{
			Name: sets.Name,
			Time: sets.CreatedTimestamp,
		}) {
			rulesets = append(rulesets, sets.Name)
		}
	}

	return rulesets, nil
}

// Deletes all receipt rules
func (s *SesReceiptRule) nukeAll(sets []*string) error {
	if len(sets) == 0 {
		logging.Debugf("No SES rule sets sets to nuke in region %s", s.Region)
		return nil
	}

	logging.Debugf("Deleting all SES rule sets in region %s", s.Region)
	var deletedSets []*string

	for _, set := range sets {

		param := &ses.DeleteReceiptRuleSetInput{
			RuleSetName: set,
		}
		_, err := s.Client.DeleteReceiptRuleSet(s.Context, param)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(set),
			ResourceType: "SES receipt rule set",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedSets = append(deletedSets, set)
			logging.Debugf("Deleted SES receipt rule set: %s", *set)
		}
	}

	logging.Debugf("[OK] %d SES receipt rule set(s) deleted in %s", len(deletedSets), s.Region)

	return nil
}

// ///////////////////////receipt Ip filters //////////////////////////////////////

// Returns a formatted string of ses-identities IDs
func (s *SesReceiptFilter) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := s.Client.ListReceiptFilters(s.Context, &ses.ListReceiptFiltersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var filters []*string
	for _, filter := range result.Filters {
		if configObj.SESReceiptFilter.ShouldInclude(config.ResourceValue{
			Name: filter.Name,
		}) {
			filters = append(filters, filter.Name)
		}
	}

	return filters, nil
}

// Deletes all filters
func (s *SesReceiptFilter) nukeAll(filters []*string) error {
	if len(filters) == 0 {
		logging.Debugf("No SES receipt filters to nuke in region %s", s.Region)
		return nil
	}

	logging.Debugf("Deleting all SES receipt  filters in region %s", s.Region)
	var deletedFilters []*string

	for _, filter := range filters {

		param := &ses.DeleteReceiptFilterInput{
			FilterName: filter,
		}
		_, err := s.Client.DeleteReceiptFilter(s.Context, param)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(filter),
			ResourceType: "SES receipt filter",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedFilters = append(deletedFilters, filter)
			logging.Debugf("Deleted SES receipt filter: %s", *filter)
		}
	}

	logging.Debugf("[OK] %d SES receipt receipt filter(s) deleted in %s", len(deletedFilters), s.Region)

	return nil
}
