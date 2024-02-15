package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of receipt rule names
func (s *SesReceiptRule) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := s.Client.ListReceiptRuleSets(&ses.ListReceiptRuleSetsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var rulesets []*string
	for _, sets := range result.RuleSets {
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
		_, err := s.Client.DeleteReceiptRuleSet(param)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(set),
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
	result, err := s.Client.ListReceiptFilters(&ses.ListReceiptFiltersInput{})
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
		_, err := s.Client.DeleteReceiptFilter(param)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(filter),
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
