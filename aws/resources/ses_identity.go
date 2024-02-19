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

// Returns a formatted string of ses-identities IDs
func (sid *SesIdentities) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	param := &ses.ListIdentitiesInput{}

	result, err := sid.Client.ListIdentities(param)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, id := range result.Identities {
		if configObj.SESIdentity.ShouldInclude(config.ResourceValue{Name: id}) {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// Deletes all identities
func (sid *SesIdentities) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No SES identities to nuke in region %s", sid.Region)
		return nil
	}

	logging.Debugf("Deleting all SES identities in region %s", sid.Region)
	var deletedIds []*string

	for _, id := range ids {
		params := &ses.DeleteIdentityInput{
			Identity: id,
		}
		_, err := sid.Client.DeleteIdentity(params)
		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "SES identity",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted SES identity: %s", *id)
		}
	}

	logging.Debugf("[OK] %d SES identity(s) deleted in %s", len(deletedIds), sid.Region)

	return nil
}
