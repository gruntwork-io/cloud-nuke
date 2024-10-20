package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (dsl *DataSyncLocation) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Data Sync Location] No Data Sync Locations found in region %s", dsl.Region)
		return nil
	}

	logging.Debugf("[Data Sync Location] Deleting all Data Sync Locations in region %s", dsl.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		logging.Debugf("[Data Sync Location] Deleting Data Sync Location %s in region %s", *identifier, dsl.Region)
		_, err := dsl.Client.DeleteLocation(dsl.Context, &datasync.DeleteLocationInput{
			LocationArn: identifier,
		})
		if err != nil {
			logging.Debugf("[Data Sync Location] Error deleting Data Sync Location %s in region %s", *identifier, dsl.Region)
			return err
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Data Sync Location] Deleted Data Sync Location %s in region %s", *identifier, dsl.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: dsl.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Data Sync Location(s) nuked in %s", len(deleted), dsl.Region)
	return nil
}

func (dsl *DataSyncLocation) getAll(c context.Context, _ config.Config) ([]*string, error) {
	var identifiers []*string
	param := &datasync.ListLocationsInput{
		MaxResults: aws.Int32(100),
	}

	locationsPaginator := datasync.NewListLocationsPaginator(dsl.Client, param)
	for locationsPaginator.HasMorePages() {
		output, err := locationsPaginator.NextPage(c)
		if err != nil {
			logging.Debugf("[Data Sync Location] Failed to list Data Sync Locations: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, location := range output.Locations {
			identifiers = append(identifiers, location.LocationArn)
		}
	}

	return identifiers, nil
}
