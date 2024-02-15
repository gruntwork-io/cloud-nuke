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

// Returns a formatted string of ses-configuartion set names
func (s *SesConfigurationSet) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	// Remove defalt route table, that will be deleted along with its TransitGateway
	param := &ses.ListConfigurationSetsInput{}

	result, err := s.Client.ListConfigurationSets(param)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var setnames []*string
	for _, set := range result.ConfigurationSets {
		if configObj.SESConfigurationSet.ShouldInclude(config.ResourceValue{Name: set.Name}) {
			setnames = append(setnames, set.Name)
		}
	}

	return setnames, nil
}

// Deletes all sets
func (s *SesConfigurationSet) nukeAll(sets []*string) error {
	if len(sets) == 0 {
		logging.Debugf("No SES configuartion sets to nuke in region %s", s.Region)
		return nil
	}

	logging.Debugf("Deleting all SES configuartion sets in region %s", s.Region)
	var deletedSets []*string

	for _, set := range sets {
		_, err := s.Client.DeleteConfigurationSet(&ses.DeleteConfigurationSetInput{
			ConfigurationSetName: set,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(set),
			ResourceType: "SES configuartion set",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedSets = append(deletedSets, set)
			logging.Debugf("Deleted SES configuartion set: %s", *set)
		}
	}

	logging.Debugf("[OK] %d SES configuartion set(s) deleted in %s", len(deletedSets), s.Region)

	return nil
}
